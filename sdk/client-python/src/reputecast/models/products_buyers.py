from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.product_count import ProductCount





T = TypeVar("T", bound="ProductsBuyers")



@_attrs_define
class ProductsBuyers:
    """ 
        Attributes:
            contacts (int): Buyers who were contacts already
            joined (int): Buyers who ticked the list offer and became contacts
            none (int): Buyers who aren't contacts
            offer_text (str):
            product (str):
            ticked (int):
            ticked_by (list[ProductCount]):
     """

    contacts: int
    joined: int
    none: int
    offer_text: str
    product: str
    ticked: int
    ticked_by: list[ProductCount]





    def to_dict(self) -> dict[str, Any]:
        from ..models.product_count import ProductCount # noqa: PLC0415
        contacts = self.contacts

        joined = self.joined

        none = self.none

        offer_text = self.offer_text

        product = self.product

        ticked = self.ticked

        ticked_by = []
        for ticked_by_item_data in self.ticked_by:
            ticked_by_item = ticked_by_item_data.to_dict()
            ticked_by.append(ticked_by_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contacts": contacts,
            "joined": joined,
            "none": none,
            "offerText": offer_text,
            "product": product,
            "ticked": ticked,
            "tickedBy": ticked_by,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.product_count import ProductCount # noqa: PLC0415
        d = dict(src_dict)
        contacts = d.pop("contacts")

        joined = d.pop("joined")

        none = d.pop("none")

        offer_text = d.pop("offerText")

        product = d.pop("product")

        ticked = d.pop("ticked")

        ticked_by = []
        _ticked_by = d.pop("tickedBy")
        for ticked_by_item_data in (_ticked_by):
            ticked_by_item = ProductCount.from_dict(ticked_by_item_data)



            ticked_by.append(ticked_by_item)


        products_buyers = cls(
            contacts=contacts,
            joined=joined,
            none=none,
            offer_text=offer_text,
            product=product,
            ticked=ticked,
            ticked_by=ticked_by,
        )

        return products_buyers

