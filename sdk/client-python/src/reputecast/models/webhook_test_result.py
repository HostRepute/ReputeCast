from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.webhook_delivery_view import WebhookDeliveryView





T = TypeVar("T", bound="WebhookTestResult")



@_attrs_define
class WebhookTestResult:
    """ 
        Attributes:
            delivery (WebhookDeliveryView):
     """

    delivery: WebhookDeliveryView





    def to_dict(self) -> dict[str, Any]:
        from ..models.webhook_delivery_view import WebhookDeliveryView # noqa: PLC0415
        delivery = self.delivery.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "delivery": delivery,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.webhook_delivery_view import WebhookDeliveryView # noqa: PLC0415
        d = dict(src_dict)
        delivery = WebhookDeliveryView.from_dict(d.pop("delivery"))




        webhook_test_result = cls(
            delivery=delivery,
        )

        return webhook_test_result

