from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="ProductDelivery")



@_attrs_define
class ProductDelivery:
    """ 
        Attributes:
            downloads (int):
            link_days (int):
            revoke_on_refund (bool): Revoke access when a payment is refunded or disputed
            template_id (str): The transactional template of the delivery email
     """

    downloads: int
    link_days: int
    revoke_on_refund: bool
    template_id: str





    def to_dict(self) -> dict[str, Any]:
        downloads = self.downloads

        link_days = self.link_days

        revoke_on_refund = self.revoke_on_refund

        template_id = self.template_id


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "downloads": downloads,
            "linkDays": link_days,
            "revokeOnRefund": revoke_on_refund,
            "templateId": template_id,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        downloads = d.pop("downloads")

        link_days = d.pop("linkDays")

        revoke_on_refund = d.pop("revokeOnRefund")

        template_id = d.pop("templateId")

        product_delivery = cls(
            downloads=downloads,
            link_days=link_days,
            revoke_on_refund=revoke_on_refund,
            template_id=template_id,
        )

        return product_delivery

