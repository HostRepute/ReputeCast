from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CouponPoolWaiting")



@_attrs_define
class CouponPoolWaiting:
    """ 
        Attributes:
            emails (int):
            id (str):
            name (str):
     """

    emails: int
    id: str
    name: str





    def to_dict(self) -> dict[str, Any]:
        emails = self.emails

        id = self.id

        name = self.name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "emails": emails,
            "id": id,
            "name": name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        emails = d.pop("emails")

        id = d.pop("id")

        name = d.pop("name")

        coupon_pool_waiting = cls(
            emails=emails,
            id=id,
            name=name,
        )

        return coupon_pool_waiting

