import re
from dataclasses import dataclass


@dataclass(frozen=True)
class Compensation:
    low: float
    high: float
    currency: str
    period: str

    def as_dict(self) -> dict:
        return {"low": self.low, "high": self.high, "currency": self.currency, "period": self.period}


_RANGE = re.compile(
    r"(?P<currency>\$|€|£|USD|EUR|GBP)\s*"
    r"(?P<low>\d[\d,]*(?:\.\d+)?)\s*(?P<low_suffix>[kK])?"
    r"(?:\s*(?:-|–|—|to)\s*(?P<currency2>\$|€|£|USD|EUR|GBP)?\s*"
    r"(?P<high>\d[\d,]*(?:\.\d+)?)\s*(?P<high_suffix>[kK])?)?",
    re.IGNORECASE,
)


def _currency(value: str) -> str:
    return {"$": "USD", "€": "EUR", "£": "GBP"}.get(value.upper(), value.upper())


def _amount(value: str, suffix: str | None) -> float:
    amount = float(value.replace(",", ""))
    return amount * 1000 if suffix else amount


def extract_compensation(text: str) -> Compensation | None:
    match = _RANGE.search(text)
    if not match:
        return None
    low = _amount(match.group("low"), match.group("low_suffix"))
    high = _amount(match.group("high") or match.group("low"), match.group("high_suffix") or match.group("low_suffix"))
    context = text[max(0, match.start() - 20):match.end() + 30].lower()
    period = "hour" if any(term in context for term in ("/hr", "/hour", "hourly", "per hour")) else "year"
    return Compensation(low, high, _currency(match.group("currency")), period)
