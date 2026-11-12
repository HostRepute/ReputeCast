from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.commerce_block_mode import CommerceBlockMode
from ..models.commerce_block_sold_out import CommerceBlockSoldOut
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID






T = TypeVar("T", bound="CommerceBlock")



@_attrs_define
class CommerceBlock:
    """ 
        Attributes:
            mode (CommerceBlockMode):
            products (list[str]):
            store (UUID):
            button (str | Unset):
            collection (str | Unset):
            compare_at (bool | Unset):
            count (int | Unset):
            heading (str | Unset):
            sold_out (CommerceBlockSoldOut | Unset):
     """

    mode: CommerceBlockMode
    products: list[str]
    store: UUID
    button: str | Unset = UNSET
    collection: str | Unset = UNSET
    compare_at: bool | Unset = UNSET
    count: int | Unset = UNSET
    heading: str | Unset = UNSET
    sold_out: CommerceBlockSoldOut | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        mode = self.mode.value

        products = self.products



        store = str(self.store)

        button = self.button

        collection = self.collection

        compare_at = self.compare_at

        count = self.count

        heading = self.heading

        sold_out: str | Unset = UNSET
        if not isinstance(self.sold_out, Unset):
            sold_out = self.sold_out.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "mode": mode,
            "products": products,
            "store": store,
        })
        if button is not UNSET:
            field_dict["button"] = button
        if collection is not UNSET:
            field_dict["collection"] = collection
        if compare_at is not UNSET:
            field_dict["compareAt"] = compare_at
        if count is not UNSET:
            field_dict["count"] = count
        if heading is not UNSET:
            field_dict["heading"] = heading
        if sold_out is not UNSET:
            field_dict["soldOut"] = sold_out

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        mode = CommerceBlockMode(d.pop("mode"))




        products = cast(list[str], d.pop("products"))


        store = UUID(d.pop("store"))




        button = d.pop("button", UNSET)

        collection = d.pop("collection", UNSET)

        compare_at = d.pop("compareAt", UNSET)

        count = d.pop("count", UNSET)

        heading = d.pop("heading", UNSET)

        _sold_out = d.pop("soldOut", UNSET)
        sold_out: CommerceBlockSoldOut | Unset
        if isinstance(_sold_out,  Unset):
            sold_out = UNSET
        else:
            sold_out = CommerceBlockSoldOut(_sold_out)




        commerce_block = cls(
            mode=mode,
            products=products,
            store=store,
            button=button,
            collection=collection,
            compare_at=compare_at,
            count=count,
            heading=heading,
            sold_out=sold_out,
        )

        return commerce_block

