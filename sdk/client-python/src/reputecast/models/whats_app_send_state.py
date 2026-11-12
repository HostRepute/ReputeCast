from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_send_state_phase import WhatsAppSendStatePhase
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppSendState")



@_attrs_define
class WhatsAppSendState:
    """ 
        Attributes:
            phase (WhatsAppSendStatePhase):
            recipients (int):
            scheduled_at (datetime.datetime):
            completed_at (datetime.datetime | Unset):
            started_by (str | Unset):
     """

    phase: WhatsAppSendStatePhase
    recipients: int
    scheduled_at: datetime.datetime
    completed_at: datetime.datetime | Unset = UNSET
    started_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        phase = self.phase.value

        recipients = self.recipients

        scheduled_at = self.scheduled_at.isoformat()

        completed_at: str | Unset = UNSET
        if not isinstance(self.completed_at, Unset):
            completed_at = self.completed_at.isoformat()

        started_by = self.started_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "phase": phase,
            "recipients": recipients,
            "scheduledAt": scheduled_at,
        })
        if completed_at is not UNSET:
            field_dict["completedAt"] = completed_at
        if started_by is not UNSET:
            field_dict["startedBy"] = started_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        phase = WhatsAppSendStatePhase(d.pop("phase"))




        recipients = d.pop("recipients")

        scheduled_at = datetime.datetime.fromisoformat(d.pop("scheduledAt"))




        _completed_at = d.pop("completedAt", UNSET)
        completed_at: datetime.datetime | Unset
        if isinstance(_completed_at,  Unset):
            completed_at = UNSET
        else:
            completed_at = datetime.datetime.fromisoformat(_completed_at)




        started_by = d.pop("startedBy", UNSET)

        whats_app_send_state = cls(
            phase=phase,
            recipients=recipients,
            scheduled_at=scheduled_at,
            completed_at=completed_at,
            started_by=started_by,
        )

        return whats_app_send_state

