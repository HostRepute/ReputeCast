from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.whats_app_send_result_status import WhatsAppSendResultStatus
from typing import cast
import datetime






T = TypeVar("T", bound="WhatsAppSendResult")



@_attrs_define
class WhatsAppSendResult:
    """ 
        Attributes:
            later (int):
            now (int):
            recipients (int):
            scheduled_at (datetime.datetime):
            status (WhatsAppSendResultStatus):
     """

    later: int
    now: int
    recipients: int
    scheduled_at: datetime.datetime
    status: WhatsAppSendResultStatus





    def to_dict(self) -> dict[str, Any]:
        later = self.later

        now = self.now

        recipients = self.recipients

        scheduled_at = self.scheduled_at.isoformat()

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "later": later,
            "now": now,
            "recipients": recipients,
            "scheduledAt": scheduled_at,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        later = d.pop("later")

        now = d.pop("now")

        recipients = d.pop("recipients")

        scheduled_at = datetime.datetime.fromisoformat(d.pop("scheduledAt"))




        status = WhatsAppSendResultStatus(d.pop("status"))




        whats_app_send_result = cls(
            later=later,
            now=now,
            recipients=recipients,
            scheduled_at=scheduled_at,
            status=status,
        )

        return whats_app_send_result

