from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.webhook_event_info import WebhookEventInfo
  from ..models.webhook_view import WebhookView





T = TypeVar("T", bound="WebhookList")



@_attrs_define
class WebhookList:
    """ 
        Attributes:
            events (list[WebhookEventInfo]):
            items (list[WebhookView]):
            max_attempts (int):
            signature_header (str):
            timestamp_header (str):
     """

    events: list[WebhookEventInfo]
    items: list[WebhookView]
    max_attempts: int
    signature_header: str
    timestamp_header: str





    def to_dict(self) -> dict[str, Any]:
        from ..models.webhook_event_info import WebhookEventInfo # noqa: PLC0415
        from ..models.webhook_view import WebhookView # noqa: PLC0415
        events = []
        for events_item_data in self.events:
            events_item = events_item_data.to_dict()
            events.append(events_item)



        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        max_attempts = self.max_attempts

        signature_header = self.signature_header

        timestamp_header = self.timestamp_header


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "events": events,
            "items": items,
            "maxAttempts": max_attempts,
            "signatureHeader": signature_header,
            "timestampHeader": timestamp_header,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_event_info import WebhookEventInfo # noqa: PLC0415
        from ..models.webhook_view import WebhookView # noqa: PLC0415
        d = dict(src_dict)
        events = []
        _events = d.pop("events")
        for events_item_data in (_events):
            events_item = WebhookEventInfo.from_dict(events_item_data)



            events.append(events_item)


        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = WebhookView.from_dict(items_item_data)



            items.append(items_item)


        max_attempts = d.pop("maxAttempts")

        signature_header = d.pop("signatureHeader")

        timestamp_header = d.pop("timestampHeader")

        webhook_list = cls(
            events=events,
            items=items,
            max_attempts=max_attempts,
            signature_header=signature_header,
            timestamp_header=timestamp_header,
        )

        return webhook_list

