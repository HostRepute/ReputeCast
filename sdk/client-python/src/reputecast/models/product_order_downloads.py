from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="ProductOrderDownloads")



@_attrs_define
class ProductOrderDownloads:
    """ 
        Attributes:
            addresses (int): Distinct IP addresses (kept only as hashes)
            count (int):
            first (datetime.datetime | Unset):
            last (datetime.datetime | Unset):
     """

    addresses: int
    count: int
    first: datetime.datetime | Unset = UNSET
    last: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        addresses = self.addresses

        count = self.count

        first: str | Unset = UNSET
        if not isinstance(self.first, Unset):
            first = self.first.isoformat()

        last: str | Unset = UNSET
        if not isinstance(self.last, Unset):
            last = self.last.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addresses": addresses,
            "count": count,
        })
        if first is not UNSET:
            field_dict["first"] = first
        if last is not UNSET:
            field_dict["last"] = last

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        addresses = d.pop("addresses")

        count = d.pop("count")

        _first = d.pop("first", UNSET)
        first: datetime.datetime | Unset
        if isinstance(_first,  Unset):
            first = UNSET
        else:
            first = datetime.datetime.fromisoformat(_first)




        _last = d.pop("last", UNSET)
        last: datetime.datetime | Unset
        if isinstance(_last,  Unset):
            last = UNSET
        else:
            last = datetime.datetime.fromisoformat(_last)




        product_order_downloads = cls(
            addresses=addresses,
            count=count,
            first=first,
            last=last,
        )

        return product_order_downloads

