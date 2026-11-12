from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.webhook_delivery_view import WebhookDeliveryView





T = TypeVar("T", bound="WebhookDeliveryDetail")



@_attrs_define
class WebhookDeliveryDetail:
    """ 
        Attributes:
            delivery (WebhookDeliveryView):
            payload (Any):
            response (str): First kilobyte of the endpoint's last response
            payload_hidden (str | Unset):
     """

    delivery: WebhookDeliveryView
    payload: Any
    response: str
    payload_hidden: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.webhook_delivery_view import WebhookDeliveryView # noqa: PLC0415
        delivery = self.delivery.to_dict()

        payload = self.payload

        response = self.response

        payload_hidden = self.payload_hidden


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "delivery": delivery,
            "payload": payload,
            "response": response,
        })
        if payload_hidden is not UNSET:
            field_dict["payloadHidden"] = payload_hidden

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_delivery_view import WebhookDeliveryView # noqa: PLC0415
        d = dict(src_dict)
        delivery = WebhookDeliveryView.from_dict(d.pop("delivery"))




        payload = d.pop("payload")

        response = d.pop("response")

        payload_hidden = d.pop("payloadHidden", UNSET)

        webhook_delivery_detail = cls(
            delivery=delivery,
            payload=payload,
            response=response,
            payload_hidden=payload_hidden,
        )

        return webhook_delivery_detail

