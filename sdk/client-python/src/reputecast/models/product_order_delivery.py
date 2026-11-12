from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductOrderDelivery")



@_attrs_define
class ProductOrderDelivery:
    """ 
        Attributes:
            new_link (bool):
            provider (str):
            seconds (float):
            status (str):
            template (str):
            version (int):
            delivered_at (datetime.datetime | Unset):
            message_id (str | Unset):
            sent_at (datetime.datetime | Unset):
     """

    new_link: bool
    provider: str
    seconds: float
    status: str
    template: str
    version: int
    delivered_at: datetime.datetime | Unset = UNSET
    message_id: str | Unset = UNSET
    sent_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        new_link = self.new_link

        provider = self.provider

        seconds = self.seconds

        status = self.status

        template = self.template

        version = self.version

        delivered_at: str | Unset = UNSET
        if not isinstance(self.delivered_at, Unset):
            delivered_at = self.delivered_at.isoformat()

        message_id = self.message_id

        sent_at: str | Unset = UNSET
        if not isinstance(self.sent_at, Unset):
            sent_at = self.sent_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "newLink": new_link,
            "provider": provider,
            "seconds": seconds,
            "status": status,
            "template": template,
            "version": version,
        })
        if delivered_at is not UNSET:
            field_dict["deliveredAt"] = delivered_at
        if message_id is not UNSET:
            field_dict["messageId"] = message_id
        if sent_at is not UNSET:
            field_dict["sentAt"] = sent_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        new_link = d.pop("newLink")

        provider = d.pop("provider")

        seconds = d.pop("seconds")

        status = d.pop("status")

        template = d.pop("template")

        version = d.pop("version")

        _delivered_at = d.pop("deliveredAt", UNSET)
        delivered_at: datetime.datetime | Unset
        if isinstance(_delivered_at,  Unset):
            delivered_at = UNSET
        else:
            delivered_at = datetime.datetime.fromisoformat(_delivered_at)




        message_id = d.pop("messageId", UNSET)

        _sent_at = d.pop("sentAt", UNSET)
        sent_at: datetime.datetime | Unset
        if isinstance(_sent_at,  Unset):
            sent_at = UNSET
        else:
            sent_at = datetime.datetime.fromisoformat(_sent_at)




        product_order_delivery = cls(
            new_link=new_link,
            provider=provider,
            seconds=seconds,
            status=status,
            template=template,
            version=version,
            delivered_at=delivered_at,
            message_id=message_id,
            sent_at=sent_at,
        )

        return product_order_delivery

