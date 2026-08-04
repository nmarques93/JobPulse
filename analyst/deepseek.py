import json
import os
import urllib.error
import urllib.request
from typing import Any


API_URL = "https://api.deepseek.com/chat/completions"


def enrich(posting: dict[str, Any], profile: dict[str, Any], baseline: dict[str, Any]) -> dict[str, Any] | None:
    """Ask DeepSeek to interpret ambiguity; deterministic analysis remains authoritative for hard gates."""
    api_key = os.getenv("DEEPSEEK_API_KEY")
    if not api_key:
        return None
    request_body = {
        "model": os.getenv("DEEPSEEK_MODEL", "deepseek-chat"),
        "temperature": 0,
        "response_format": {"type": "json_object"},
        "messages": [
            {
                "role": "system",
                "content": (
                    "You analyze job postings against a candidate profile. Return only valid JSON with keys: "
                    "role_type, fit_score (integer 1-10), recommendation (apply/review/skip), evidence (array), "
                    "concerns (array), questions_to_verify (array), tailored_summary. "
                    "Do not invent facts. Treat the deterministic baseline's hard-gate concerns as authoritative."
                ),
            },
            {
                "role": "user",
                "content": json.dumps({"profile": profile, "baseline": baseline, "posting": posting}),
            },
        ],
    }
    request = urllib.request.Request(
        API_URL,
        data=json.dumps(request_body).encode("utf-8"),
        headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
        method="POST",
    )
    try:
        with urllib.request.urlopen(request, timeout=30) as response:
            payload = json.loads(response.read().decode("utf-8"))
        content = payload["choices"][0]["message"]["content"]
        result = json.loads(content)
        if not isinstance(result, dict):
            raise ValueError("DeepSeek response was not a JSON object")
        return result
    except (urllib.error.URLError, TimeoutError, KeyError, IndexError, TypeError, ValueError, json.JSONDecodeError) as error:
        raise RuntimeError(f"DeepSeek enrichment failed: {error}") from error
