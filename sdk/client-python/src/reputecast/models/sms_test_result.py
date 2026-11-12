from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.sms_test_result_status import SmsTestResultStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="SmsTestResult")



@_attrs_define
class SmsTestResult:
    """ 
        Attributes:
            at (datetime.datetime):
            message_id (str):
            provider (str):
            sender (str):
            sender_id (bool):
            status (SmsTestResultStatus):
            delivered (datetime.datetime | Unset):
            seconds (float | Unset):
     """

    at: datetime.datetime
    message_id: str
    provider: str
    sender: str
    sender_id: bool
    status: SmsTestResultStatus
    delivered: datetime.datetime | Unset = UNSET
    seconds: float | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        message_id = self.message_id

        provider = self.provider

        sender = self.sender

        sender_id = self.sender_id

        status = self.status.value

        delivered: str | Unset = UNSET
        if not isinstance(self.delivered, Unset):
            delivered = self.delivered.isoformat()

        seconds = self.seconds


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "messageId": message_id,
            "provider": provider,
            "sender": sender,
            "senderId": sender_id,
            "status": status,
        })
        if delivered is not UNSET:
            field_dict["delivered"] = delivered
        if seconds is not UNSET:
            field_dict["seconds"] = seconds

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        message_id = d.pop("messageId")

        provider = d.pop("provider")

        sender = d.pop("sender")

        sender_id = d.pop("senderId")

        status = SmsTestResultStatus(d.pop("status"))




        _delivered = d.pop("delivered", UNSET)
        delivered: datetime.datetime | Unset
        if isinstance(_delivered,  Unset):
            delivered = UNSET
        else:
            delivered = datetime.datetime.fromisoformat(_delivered)




        seconds = d.pop("seconds", UNSET)

        sms_test_result = cls(
            at=at,
            message_id=message_id,
            provider=provider,
            sender=sender,
            sender_id=sender_id,
            status=status,
            delivered=delivered,
            seconds=seconds,
        )

        return sms_test_result

