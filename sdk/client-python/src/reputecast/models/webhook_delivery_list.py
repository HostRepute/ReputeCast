from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.webhook_delivery_list_queued import WebhookDeliveryListQueued
  from ..models.webhook_delivery_view import WebhookDeliveryView





T = TypeVar("T", bound="WebhookDeliveryList")



@_attrs_define
class WebhookDeliveryList:
    """ 
        Attributes:
            items (list[WebhookDeliveryView]):
            queued (WebhookDeliveryListQueued):
     """

    items: list[WebhookDeliveryView]
    queued: WebhookDeliveryListQueued





    def to_dict(self) -> dict[str, Any]:
        from ..models.webhook_delivery_list_queued import WebhookDeliveryListQueued # noqa: PLC0415
        from ..models.webhook_delivery_view import WebhookDeliveryView # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        queued = self.queued.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "queued": queued,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_delivery_list_queued import WebhookDeliveryListQueued # noqa: PLC0415
        from ..models.webhook_delivery_view import WebhookDeliveryView # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = WebhookDeliveryView.from_dict(items_item_data)



            items.append(items_item)


        queued = WebhookDeliveryListQueued.from_dict(d.pop("queued"))




        webhook_delivery_list = cls(
            items=items,
            queued=queued,
        )

        return webhook_delivery_list

