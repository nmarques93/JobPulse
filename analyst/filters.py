import re
from dataclasses import dataclass, field
from typing import Any


def _terms(profile: dict[str, Any], *keys: str) -> list[str]:
    values: list[str] = []
    for key in keys:
        values.extend(profile.get(key, []) or [])
    return [str(value).strip().lower() for value in values if str(value).strip()]


def _matches(term: str, text: str) -> bool:
    return bool(re.search(r"\b" + re.escape(term) + r"\b", text))


@dataclass(frozen=True)
class Match:
    score: int
    matched: list[str]
    gaps: list[str]
    recommendation: str
    role_type: str = "unknown"
    evidence: list[str] = field(default_factory=list)
    concerns: list[str] = field(default_factory=list)


def score_posting(title: str, location: str, description: str, profile: dict[str, Any]) -> Match:
    title_text = title.lower()
    location_text = location.lower()
    text = " ".join((title, location, description)).lower()
    role_types = profile.get("role_types", {})
    excluded_role_types = _terms(profile, "excluded_role_types")
    excluded_keywords = _terms(profile, "excluded_keywords")
    required_keywords = _terms(profile, "required_keywords")
    eligible_locations = _terms(profile, "eligible_locations")
    remote_terms = _terms(profile, "remote_terms") or ["remote", "distributed"]
    preferred_domains = _terms(profile, "preferred_domains")
    strong_skills = _terms(profile, "strong_skills")
    transferable_skills = _terms(profile, "transferable_skills")
    seniority = _terms(profile, "seniority")

    role_type = "unknown"
    role_matches: list[str] = []
    for role, terms in role_types.items():
        matches = [term.lower() for term in terms if _matches(term.lower(), title_text)]
        if matches:
            role_type = role
            role_matches = matches
            break

    matched = [skill for skill in strong_skills + transferable_skills if _matches(skill, text)]
    gaps = [skill for skill in strong_skills if skill not in matched]
    domain_matches = [domain for domain in preferred_domains if _matches(domain, text)]
    seniority_match = any(_matches(term, title_text) for term in seniority)
    location_match = (
        not eligible_locations
        or any(_matches(term, location_text) for term in eligible_locations)
        or (profile.get("remote_allowed", True) and any(_matches(term, text) for term in remote_terms))
    )
    excluded = [term for term in excluded_keywords if _matches(term, text)]
    excluded_roles = [role for role in excluded_role_types if _matches(role, title_text)]
    missing_required = [term for term in required_keywords if not _matches(term, text)]

    concerns: list[str] = []
    evidence: list[str] = []
    if role_matches:
        evidence.append(f"Role matches {role_type}: {', '.join(role_matches)}")
    if matched:
        evidence.append(f"Relevant skills: {', '.join(matched)}")
    if domain_matches:
        evidence.append(f"Preferred domain: {', '.join(domain_matches)}")
    if seniority_match:
        evidence.append("Seniority matches the configured target levels")
    if not location_match:
        concerns.append("Location or remote eligibility is not compatible")
    if not seniority_match and seniority:
        concerns.append("Target seniority was not found in the job title")
    if missing_required:
        concerns.append(f"Missing required terms: {', '.join(missing_required)}")
    if excluded:
        concerns.append(f"Excluded terms found: {', '.join(excluded)}")
    if excluded_roles:
        concerns.append(f"Excluded role type found: {', '.join(excluded_roles)}")
    if not location_match or excluded or excluded_roles or missing_required:
        recommendation = "skip"
    else:
        raw = 0.0
        raw += 2.5 if role_matches else 0
        raw += min(2.5, len(matched) * 0.5)
        raw += 1.5 if domain_matches else 0
        raw += 1.5 if seniority_match else 0
        raw += 2.0 if location_match else 0
        score = min(10, max(1, round(raw)))
        recommendation = "review" if score >= int(profile.get("minimum_score", 5)) else "skip"
    if "score" not in locals():
        score = 1
    return Match(score, matched, gaps, recommendation, role_type, evidence, concerns)
