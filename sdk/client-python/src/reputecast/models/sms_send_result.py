from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_send_result_status import SmsSendResultStatus
from typing import cast
import datetime






T = TypeVar("T", bound="SmsSendResult")



@_attrs_define
class SmsSendResult:
    """ 
        Attributes:
            recipients (int):
            scheduled_at (datetime.datetime):
            status (SmsSendResultStatus):
     """

    recipients: int
    scheduled_at: datetime.datetime
    status: SmsSendResultStatus





    def to_dict(self) -> dict[str, Any]:
        recipients = self.recipients

        scheduled_at = self.scheduled_at.isoformat()

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "recipients": recipients,
            "scheduledAt": scheduled_at,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        recipients = d.pop("recipients")

        scheduled_at = datetime.datetime.fromisoformat(d.pop("scheduledAt"))




        status = SmsSendResultStatus(d.pop("status"))




        sms_send_result = cls(
            recipients=recipients,
            scheduled_at=scheduled_at,
            status=status,
        )

        return sms_send_result

