from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.live_threshold_key import LiveThresholdKey
from ..models.live_threshold_reason import LiveThresholdReason
from ..models.live_threshold_state import LiveThresholdState
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.live_threshold_part import LiveThresholdPart





T = TypeVar("T", bound="LiveThreshold")



@_attrs_define
class LiveThreshold:
    """ 
        Attributes:
            any_volume (bool):
            base (int):
            events (int):
            key (LiveThresholdKey):
            limit (float):
            min_base (int):
            min_events (int):
            state (LiveThresholdState):
            action (str | Unset):
            duration (int | Unset):
            mailbox (str | Unset):
            parts (list[LiveThresholdPart] | Unset):
            reason (LiveThresholdReason | Unset):
            value (float | Unset):
            window (int | Unset):
     """

    any_volume: bool
    base: int
    events: int
    key: LiveThresholdKey
    limit: float
    min_base: int
    min_events: int
    state: LiveThresholdState
    action: str | Unset = UNSET
    duration: int | Unset = UNSET
    mailbox: str | Unset = UNSET
    parts: list[LiveThresholdPart] | Unset = UNSET
    reason: LiveThresholdReason | Unset = UNSET
    value: float | Unset = UNSET
    window: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.live_threshold_part import LiveThresholdPart # noqa: PLC0415
        any_volume = self.any_volume

        base = self.base

        events = self.events

        key = self.key.value

        limit = self.limit

        min_base = self.min_base

        min_events = self.min_events

        state = self.state.value

        action = self.action

        duration = self.duration

        mailbox = self.mailbox

        parts: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.parts, Unset):
            parts = []
            for parts_item_data in self.parts:
                parts_item = parts_item_data.to_dict()
                parts.append(parts_item)



        reason: str | Unset = UNSET
        if not isinstance(self.reason, Unset):
            reason = self.reason.value


        value = self.value

        window = self.window


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "anyVolume": any_volume,
            "base": base,
            "events": events,
            "key": key,
            "limit": limit,
            "minBase": min_base,
            "minEvents": min_events,
            "state": state,
        })
        if action is not UNSET:
            field_dict["action"] = action
        if duration is not UNSET:
            field_dict["duration"] = duration
        if mailbox is not UNSET:
            field_dict["mailbox"] = mailbox
        if parts is not UNSET:
            field_dict["parts"] = parts
        if reason is not UNSET:
            field_dict["reason"] = reason
        if value is not UNSET:
            field_dict["value"] = value
        if window is not UNSET:
            field_dict["window"] = window

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.live_threshold_part import LiveThresholdPart # noqa: PLC0415
        d = dict(src_dict)
        any_volume = d.pop("anyVolume")

        base = d.pop("base")

        events = d.pop("events")

        key = LiveThresholdKey(d.pop("key"))




        limit = d.pop("limit")

        min_base = d.pop("minBase")

        min_events = d.pop("minEvents")

        state = LiveThresholdState(d.pop("state"))




        action = d.pop("action", UNSET)

        duration = d.pop("duration", UNSET)

        mailbox = d.pop("mailbox", UNSET)

        _parts = d.pop("parts", UNSET)
        parts: list[LiveThresholdPart] | Unset = UNSET
        if _parts is not UNSET:
            parts = []
            for parts_item_data in _parts:
                parts_item = LiveThresholdPart.from_dict(parts_item_data)



                parts.append(parts_item)


        _reason = d.pop("reason", UNSET)
        reason: LiveThresholdReason | Unset
        if isinstance(_reason,  Unset):
            reason = UNSET
        else:
            reason = LiveThresholdReason(_reason)




        value = d.pop("value", UNSET)

        window = d.pop("window", UNSET)

        live_threshold = cls(
            any_volume=any_volume,
            base=base,
            events=events,
            key=key,
            limit=limit,
            min_base=min_base,
            min_events=min_events,
            state=state,
            action=action,
            duration=duration,
            mailbox=mailbox,
            parts=parts,
            reason=reason,
            value=value,
            window=window,
        )

        return live_threshold

