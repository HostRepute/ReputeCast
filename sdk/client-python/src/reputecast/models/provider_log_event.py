from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_log_event_channel import ProviderLogEventChannel
from ..models.provider_log_event_type import ProviderLogEventType
from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProviderLogEvent")



@_attrs_define
class ProviderLogEvent:
    """ 
        Attributes:
            bounce_class (str):
            channel (ProviderLogEventChannel):
            code (str):
            diagnostic (str):
            id (str):
            occurred_at (datetime.datetime):
            provider (str):
            provider_message_id (str):
            raw_type (str):
            received_at (datetime.datetime):
            recipient (str): Masked
            source (str):
            type_ (ProviderLogEventType):
            campaign_id (str | Unset):
            provider_id (str | Unset):
     """

    bounce_class: str
    channel: ProviderLogEventChannel
    code: str
    diagnostic: str
    id: str
    occurred_at: datetime.datetime
    provider: str
    provider_message_id: str
    raw_type: str
    received_at: datetime.datetime
    recipient: str
    source: str
    type_: ProviderLogEventType
    campaign_id: str | Unset = UNSET
    provider_id: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        bounce_class = self.bounce_class

        channel = self.channel.value

        code = self.code

        diagnostic = self.diagnostic

        id = self.id

        occurred_at = self.occurred_at.isoformat()

        provider = self.provider

        provider_message_id = self.provider_message_id

        raw_type = self.raw_type

        received_at = self.received_at.isoformat()

        recipient = self.recipient

        source = self.source

        type_ = self.type_.value

        campaign_id = self.campaign_id

        provider_id = self.provider_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounceClass": bounce_class,
            "channel": channel,
            "code": code,
            "diagnostic": diagnostic,
            "id": id,
            "occurredAt": occurred_at,
            "provider": provider,
            "providerMessageId": provider_message_id,
            "rawType": raw_type,
            "receivedAt": received_at,
            "recipient": recipient,
            "source": source,
            "type": type_,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if provider_id is not UNSET:
            field_dict["providerId"] = provider_id

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        bounce_class = d.pop("bounceClass")

        channel = ProviderLogEventChannel(d.pop("channel"))




        code = d.pop("code")

        diagnostic = d.pop("diagnostic")

        id = d.pop("id")

        occurred_at = datetime.datetime.fromisoformat(d.pop("occurredAt"))




        provider = d.pop("provider")

        provider_message_id = d.pop("providerMessageId")

        raw_type = d.pop("rawType")

        received_at = datetime.datetime.fromisoformat(d.pop("receivedAt"))




        recipient = d.pop("recipient")

        source = d.pop("source")

        type_ = ProviderLogEventType(d.pop("type"))




        campaign_id = d.pop("campaignId", UNSET)

        provider_id = d.pop("providerId", UNSET)

        provider_log_event = cls(
            bounce_class=bounce_class,
            channel=channel,
            code=code,
            diagnostic=diagnostic,
            id=id,
            occurred_at=occurred_at,
            provider=provider,
            provider_message_id=provider_message_id,
            raw_type=raw_type,
            received_at=received_at,
            recipient=recipient,
            source=source,
            type_=type_,
            campaign_id=campaign_id,
            provider_id=provider_id,
        )

        return provider_log_event

