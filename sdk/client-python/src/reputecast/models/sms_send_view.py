from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_send_view_paused_kind import SmsSendViewPausedKind
from ..models.sms_send_view_phase import SmsSendViewPhase
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsSendView")



@_attrs_define
class SmsSendView:
    """ 
        Attributes:
            paused_kind (SmsSendViewPausedKind):
            phase (SmsSendViewPhase):
            scheduled_at (datetime.datetime):
            started_by (str):
            paused_at (datetime.datetime | Unset):
     """

    paused_kind: SmsSendViewPausedKind
    phase: SmsSendViewPhase
    scheduled_at: datetime.datetime
    started_by: str
    paused_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        paused_kind = self.paused_kind.value

        phase = self.phase.value

        scheduled_at = self.scheduled_at.isoformat()

        started_by = self.started_by

        paused_at: str | Unset = UNSET
        if not isinstance(self.paused_at, Unset):
            paused_at = self.paused_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "pausedKind": paused_kind,
            "phase": phase,
            "scheduledAt": scheduled_at,
            "startedBy": started_by,
        })
        if paused_at is not UNSET:
            field_dict["pausedAt"] = paused_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        paused_kind = SmsSendViewPausedKind(d.pop("pausedKind"))




        phase = SmsSendViewPhase(d.pop("phase"))




        scheduled_at = datetime.datetime.fromisoformat(d.pop("scheduledAt"))




        started_by = d.pop("startedBy")

        _paused_at = d.pop("pausedAt", UNSET)
        paused_at: datetime.datetime | Unset
        if isinstance(_paused_at,  Unset):
            paused_at = UNSET
        else:
            paused_at = datetime.datetime.fromisoformat(_paused_at)




        sms_send_view = cls(
            paused_kind=paused_kind,
            phase=phase,
            scheduled_at=scheduled_at,
            started_by=started_by,
            paused_at=paused_at,
        )

        return sms_send_view

