from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime






T = TypeVar("T", bound="CommerceStoreDiscounts")



@_attrs_define
class CommerceStoreDiscounts:
    """ 
        Attributes:
            granted (bool):
            pools (list[str]):
            supported (bool):
            allowed_at (datetime.datetime | Unset):
            allowed_by (str | Unset):
     """

    granted: bool
    pools: list[str]
    supported: bool
    allowed_at: datetime.datetime | Unset = UNSET
    allowed_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        granted = self.granted

        pools = self.pools



        supported = self.supported

        allowed_at: str | Unset = UNSET
        if not isinstance(self.allowed_at, Unset):
            allowed_at = self.allowed_at.isoformat()

        allowed_by = self.allowed_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "granted": granted,
            "pools": pools,
            "supported": supported,
        })
        if allowed_at is not UNSET:
            field_dict["allowedAt"] = allowed_at
        if allowed_by is not UNSET:
            field_dict["allowedBy"] = allowed_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        granted = d.pop("granted")

        pools = cast(list[str], d.pop("pools"))


        supported = d.pop("supported")

        _allowed_at = d.pop("allowedAt", UNSET)
        allowed_at: datetime.datetime | Unset
        if isinstance(_allowed_at,  Unset):
            allowed_at = UNSET
        else:
            allowed_at = datetime.datetime.fromisoformat(_allowed_at)




        allowed_by = d.pop("allowedBy", UNSET)

        commerce_store_discounts = cls(
            granted=granted,
            pools=pools,
            supported=supported,
            allowed_at=allowed_at,
            allowed_by=allowed_by,
        )

        return commerce_store_discounts

