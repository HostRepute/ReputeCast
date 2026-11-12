from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast
import datetime






T = TypeVar("T", bound="PushPromptStats")



@_attrs_define
class PushPromptStats:
    """ 
        Attributes:
            allowed (int):
            allowed_pct (float):
            bell_clicked (int):
            blocked (int):
            closed (int):
            closed_pct (float):
            granted (int):
            granted_pct (float):
            ios_added (int):
            ios_added_pct (float):
            ios_steps (int):
            ios_subscribers (int):
            not_now (int):
            not_now_pct (float):
            shown (int):
            since (datetime.datetime):
            since_snippet (bool):
     """

    allowed: int
    allowed_pct: float
    bell_clicked: int
    blocked: int
    closed: int
    closed_pct: float
    granted: int
    granted_pct: float
    ios_added: int
    ios_added_pct: float
    ios_steps: int
    ios_subscribers: int
    not_now: int
    not_now_pct: float
    shown: int
    since: datetime.datetime
    since_snippet: bool





    def to_dict(self) -> dict[str, Any]:
        allowed = self.allowed

        allowed_pct = self.allowed_pct

        bell_clicked = self.bell_clicked

        blocked = self.blocked

        closed = self.closed

        closed_pct = self.closed_pct

        granted = self.granted

        granted_pct = self.granted_pct

        ios_added = self.ios_added

        ios_added_pct = self.ios_added_pct

        ios_steps = self.ios_steps

        ios_subscribers = self.ios_subscribers

        not_now = self.not_now

        not_now_pct = self.not_now_pct

        shown = self.shown

        since = self.since.isoformat()

        since_snippet = self.since_snippet


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "allowed": allowed,
            "allowedPct": allowed_pct,
            "bellClicked": bell_clicked,
            "blocked": blocked,
            "closed": closed,
            "closedPct": closed_pct,
            "granted": granted,
            "grantedPct": granted_pct,
            "iosAdded": ios_added,
            "iosAddedPct": ios_added_pct,
            "iosSteps": ios_steps,
            "iosSubscribers": ios_subscribers,
            "notNow": not_now,
            "notNowPct": not_now_pct,
            "shown": shown,
            "since": since,
            "sinceSnippet": since_snippet,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        allowed = d.pop("allowed")

        allowed_pct = d.pop("allowedPct")

        bell_clicked = d.pop("bellClicked")

        blocked = d.pop("blocked")

        closed = d.pop("closed")

        closed_pct = d.pop("closedPct")

        granted = d.pop("granted")

        granted_pct = d.pop("grantedPct")

        ios_added = d.pop("iosAdded")

        ios_added_pct = d.pop("iosAddedPct")

        ios_steps = d.pop("iosSteps")

        ios_subscribers = d.pop("iosSubscribers")

        not_now = d.pop("notNow")

        not_now_pct = d.pop("notNowPct")

        shown = d.pop("shown")

        since = datetime.datetime.fromisoformat(d.pop("since"))




        since_snippet = d.pop("sinceSnippet")

        push_prompt_stats = cls(
            allowed=allowed,
            allowed_pct=allowed_pct,
            bell_clicked=bell_clicked,
            blocked=blocked,
            closed=closed,
            closed_pct=closed_pct,
            granted=granted,
            granted_pct=granted_pct,
            ios_added=ios_added,
            ios_added_pct=ios_added_pct,
            ios_steps=ios_steps,
            ios_subscribers=ios_subscribers,
            not_now=not_now,
            not_now_pct=not_now_pct,
            shown=shown,
            since=since,
            since_snippet=since_snippet,
        )

        return push_prompt_stats

