from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.warmup_settings_days import WarmupSettingsDays
from ..models.warmup_settings_guard_action import WarmupSettingsGuardAction
from ..types import UNSET, Unset






T = TypeVar("T", bound="WarmupSettings")



@_attrs_define
class WarmupSettings:
    """ 
        Attributes:
            days (WarmupSettingsDays):
            engaged_first (bool):
            guard_action (WarmupSettingsGuardAction):
            guard_bounce (float):
            guard_complaint (float):
            spread_mailbox (bool):
            overflow (bool | Unset):
     """

    days: WarmupSettingsDays
    engaged_first: bool
    guard_action: WarmupSettingsGuardAction
    guard_bounce: float
    guard_complaint: float
    spread_mailbox: bool
    overflow: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        days = self.days.value

        engaged_first = self.engaged_first

        guard_action = self.guard_action.value

        guard_bounce = self.guard_bounce

        guard_complaint = self.guard_complaint

        spread_mailbox = self.spread_mailbox

        overflow = self.overflow


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "days": days,
            "engagedFirst": engaged_first,
            "guardAction": guard_action,
            "guardBounce": guard_bounce,
            "guardComplaint": guard_complaint,
            "spreadMailbox": spread_mailbox,
        })
        if overflow is not UNSET:
            field_dict["overflow"] = overflow

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        days = WarmupSettingsDays(d.pop("days"))




        engaged_first = d.pop("engagedFirst")

        guard_action = WarmupSettingsGuardAction(d.pop("guardAction"))




        guard_bounce = d.pop("guardBounce")

        guard_complaint = d.pop("guardComplaint")

        spread_mailbox = d.pop("spreadMailbox")

        overflow = d.pop("overflow", UNSET)

        warmup_settings = cls(
            days=days,
            engaged_first=engaged_first,
            guard_action=guard_action,
            guard_bounce=guard_bounce,
            guard_complaint=guard_complaint,
            spread_mailbox=spread_mailbox,
            overflow=overflow,
        )

        return warmup_settings

