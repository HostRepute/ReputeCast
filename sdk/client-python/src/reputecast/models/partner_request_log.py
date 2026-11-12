from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.partner_request_log_direction import PartnerRequestLogDirection
from typing import cast

if TYPE_CHECKING:
  from ..models.partner_request_item import PartnerRequestItem





T = TypeVar("T", bound="PartnerRequestLog")



@_attrs_define
class PartnerRequestLog:
    """ 
        Attributes:
            direction (PartnerRequestLogDirection):
            items (list[PartnerRequestItem]):
            link_id (str):
            partner (str):
            total (int):
     """

    direction: PartnerRequestLogDirection
    items: list[PartnerRequestItem]
    link_id: str
    partner: str
    total: int





    def to_dict(self) -> dict[str, Any]:
        from ..models.partner_request_item import PartnerRequestItem # noqa: PLC0415
        direction = self.direction.value

        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        link_id = self.link_id

        partner = self.partner

        total = self.total


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "direction": direction,
            "items": items,
            "linkId": link_id,
            "partner": partner,
            "total": total,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.partner_request_item import PartnerRequestItem # noqa: PLC0415
        d = dict(src_dict)
        direction = PartnerRequestLogDirection(d.pop("direction"))




        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = PartnerRequestItem.from_dict(items_item_data)



            items.append(items_item)


        link_id = d.pop("linkId")

        partner = d.pop("partner")

        total = d.pop("total")

        partner_request_log = cls(
            direction=direction,
            items=items,
            link_id=link_id,
            partner=partner,
            total=total,
        )

        return partner_request_log

