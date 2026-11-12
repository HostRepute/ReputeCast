from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.traffic_reply_rule_action import TrafficReplyRuleAction
from ..models.traffic_reply_rule_mailbox import TrafficReplyRuleMailbox
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="TrafficReplyRule")



@_attrs_define
class TrafficReplyRule:
    """ 
        Attributes:
            action (TrafficReplyRuleAction):
            contains (str): Matched as plain text, not case-sensitive, anywhere in the reply
            count (int):
            duration_minutes (int):
            enabled (bool):
            mailbox (TrafficReplyRuleMailbox):
            window_minutes (int):
            id (str | Unset):
            last_campaign (str | Unset):
            last_matched_at (datetime.datetime | Unset):
            means (str | Unset):
            note (str | Unset): What the reply means, in the rule's own words
     """

    action: TrafficReplyRuleAction
    contains: str
    count: int
    duration_minutes: int
    enabled: bool
    mailbox: TrafficReplyRuleMailbox
    window_minutes: int
    id: str | Unset = UNSET
    last_campaign: str | Unset = UNSET
    last_matched_at: datetime.datetime | Unset = UNSET
    means: str | Unset = UNSET
    note: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        action = self.action.value

        contains = self.contains

        count = self.count

        duration_minutes = self.duration_minutes

        enabled = self.enabled

        mailbox = self.mailbox.value

        window_minutes = self.window_minutes

        id = self.id

        last_campaign = self.last_campaign

        last_matched_at: str | Unset = UNSET
        if not isinstance(self.last_matched_at, Unset):
            last_matched_at = self.last_matched_at.isoformat()

        means = self.means

        note = self.note


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "action": action,
            "contains": contains,
            "count": count,
            "durationMinutes": duration_minutes,
            "enabled": enabled,
            "mailbox": mailbox,
            "windowMinutes": window_minutes,
        })
        if id is not UNSET:
            field_dict["id"] = id
        if last_campaign is not UNSET:
            field_dict["lastCampaign"] = last_campaign
        if last_matched_at is not UNSET:
            field_dict["lastMatchedAt"] = last_matched_at
        if means is not UNSET:
            field_dict["means"] = means
        if note is not UNSET:
            field_dict["note"] = note

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        action = TrafficReplyRuleAction(d.pop("action"))




        contains = d.pop("contains")

        count = d.pop("count")

        duration_minutes = d.pop("durationMinutes")

        enabled = d.pop("enabled")

        mailbox = TrafficReplyRuleMailbox(d.pop("mailbox"))




        window_minutes = d.pop("windowMinutes")

        id = d.pop("id", UNSET)

        last_campaign = d.pop("lastCampaign", UNSET)

        _last_matched_at = d.pop("lastMatchedAt", UNSET)
        last_matched_at: datetime.datetime | Unset
        if isinstance(_last_matched_at,  Unset):
            last_matched_at = UNSET
        else:
            last_matched_at = datetime.datetime.fromisoformat(_last_matched_at)




        means = d.pop("means", UNSET)

        note = d.pop("note", UNSET)

        traffic_reply_rule = cls(
            action=action,
            contains=contains,
            count=count,
            duration_minutes=duration_minutes,
            enabled=enabled,
            mailbox=mailbox,
            window_minutes=window_minutes,
            id=id,
            last_campaign=last_campaign,
            last_matched_at=last_matched_at,
            means=means,
            note=note,
        )

        return traffic_reply_rule

