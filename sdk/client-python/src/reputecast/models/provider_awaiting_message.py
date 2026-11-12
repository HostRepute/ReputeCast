from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_awaiting_message_status import ProviderAwaitingMessageStatus
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderAwaitingMessage")



@_attrs_define
class ProviderAwaitingMessage:
    """ 
        Attributes:
            at (datetime.datetime):
            provider_message_id (str):
            recipient (str): Masked
            source (str):
            status (ProviderAwaitingMessageStatus):
            delivered_at (datetime.datetime | Unset):
     """

    at: datetime.datetime
    provider_message_id: str
    recipient: str
    source: str
    status: ProviderAwaitingMessageStatus
    delivered_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        at = self.at.isoformat()

        provider_message_id = self.provider_message_id

        recipient = self.recipient

        source = self.source

        status = self.status.value

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "at": at,
            "providerMessageId": provider_message_id,
            "recipient": recipient,
            "source": source,
            "status": status,
        })
        if delivered_at is not UNSET:
            field_dict["deliveredAt"] = delivered_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        at = datetime.datetime.fromisoformat(d.pop("at"))




        provider_message_id = d.pop("providerMessageId")

        recipient = d.pop("recipient")

        source = d.pop("source")

        status = ProviderAwaitingMessageStatus(d.pop("status"))




        _delivered_at = d.pop("deliveredAt", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at,  Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)




        provider_awaiting_message = cls(
            at=at,
            provider_message_id=provider_message_id,
            recipient=recipient,
            source=source,
            status=status,
            delivered_at=delivered_at,
        )

        return provider_awaiting_message

