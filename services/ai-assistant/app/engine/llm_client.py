import json
import logging
import re

import httpx

from app.config import config

logger = logging.getLogger(__name__)

PERFORMANCE_QA_PROMPT = """You are a trading performance analyst for a Polymarket prediction market bot. Answer the user's question using ONLY the data provided.

User question: {question}

Verified data from database:
{data_points}

Rules:
1. Use ONLY the numbers provided — do not invent or estimate
2. Reference specific data points in your answer
3. Be concise and direct
4. If the data doesn't answer the question, say so
5. Use plain English — no technical jargon
"""

DECISION_EXPLAIN_PROMPT = """You are a trading bot explainer. Explain why the bot made this trade in simple terms.

Trade details:
- Market: {market_id}
- Side: {side}
- Entry price: {entry_price}
- Exit price: {exit_price}
- PnL: {pnl}
- Strategy: {strategy_id}
- Timestamp: {fill_timestamp}

Decision context:
{decision_context}

Rules:
1. Explain in plain English — no technical jargon
2. Focus on WHY, not WHAT
3. Reference the specific data points
4. If there were risk events, explain their impact
5. Keep it under 200 words
"""

INTENT_SYSTEM_PROMPT = """You are an intent classifier for a trading bot assistant.
Classify the user's question into one of these categories:
- pnl: Questions about profit/loss, earnings, money made/lost
- win_rate: Questions about win rate, success rate, accuracy
- trade_count: Questions about number of trades, activity
- strategy: Questions about specific strategies
- market: Questions about specific markets
- explanation: Questions about why a trade was made

Return ONLY a JSON object with "category" and optional "period" or "market_id" fields.
Example: {"category": "pnl", "period": "week"}"""

JSON_PATTERN = re.compile(r"\{(?:[^{}]|\{[^{}]*\})*\}")


class LLMClient:
    def __init__(self):
        self.api_key = config.LLM_API_KEY
        self.base_url = config.LLM_BASE_URL
        self.model = config.LLM_MODEL
        self.client = httpx.AsyncClient(timeout=httpx.Timeout(connect=5.0, read=30.0, write=5.0, pool=5.0))

    async def close(self):
        """Close the HTTP client."""
        await self.client.aclose()

    async def chat_completion(self, messages: list[dict]) -> str:
        """Send chat completion request to LLM API."""
        response = await self.client.post(
            f"{self.base_url}/chat/completions",
            headers={"Authorization": f"Bearer {self.api_key}"},
            json={"model": self.model, "messages": messages, "temperature": 0.1},
        )
        response.raise_for_status()

        data = response.json()
        choices = data.get("choices", [])
        if not choices:
            raise ValueError("LLM returned empty choices")

        content = choices[0].get("message", {}).get("content")
        if content is None:
            raise ValueError("LLM returned missing content")

        return content

    async def ask_performance_question(self, question: str, data_points: list[dict]) -> str:
        """Ask a performance question with verified data."""
        if not data_points:
            return "No trading data available to answer this question."
        data_str = "\n".join(f"- {dp['label']}: {dp['value']}" for dp in data_points)

        messages = [
            {
                "role": "system",
                "content": "You are a helpful trading performance analyst. Be concise and accurate.",
            },
            {
                "role": "user",
                "content": PERFORMANCE_QA_PROMPT.format(
                    question=question,
                    data_points=data_str,
                ),
            },
        ]

        return await self.chat_completion(messages)

    async def explain_trade_decision(self, trade: dict, decision_context: str) -> str:
        """Explain why a trade was made."""
        messages = [
            {
                "role": "system",
                "content": "You are a helpful trading bot explainer. Explain decisions in simple terms.",
            },
            {
                "role": "user",
                "content": DECISION_EXPLAIN_PROMPT.format(
                    market_id=trade.get("market_id", "unknown"),
                    side=trade.get("side", "unknown"),
                    entry_price=trade.get("entry_price", "N/A"),
                    exit_price=trade.get("exit_price", "N/A"),
                    pnl=trade.get("pnl", "N/A"),
                    strategy_id=trade.get("strategy_id", "unknown"),
                    fill_timestamp=trade.get("fill_timestamp", "N/A"),
                    decision_context=decision_context,
                ),
            },
        ]

        return await self.chat_completion(messages)

    async def detect_intent(self, question: str) -> dict:
        """Detect what the user is asking about."""
        messages = [
            {"role": "system", "content": INTENT_SYSTEM_PROMPT},
            {"role": "user", "content": question},
        ]

        response = await self.chat_completion(messages)

        try:
            # Extract JSON from response (handles markdown code fences)
            match = JSON_PATTERN.search(response)
            if match:
                return json.loads(match.group())
            return json.loads(response.strip())
        except (json.JSONDecodeError, ValueError):
            logger.warning("failed to parse intent", extra={"response": response})
            return {"category": "general"}
