from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="CouponPoolTokens")



@_attrs_define
class CouponPoolTokens:
    """ 
        Attributes:
            code (str):  Example: {{ coupons.welcome15.code }}.
            url (str):  Example: {{ coupons.welcome15.url }}.
     """

    code: str
    url: str





    def to_dict(self) -> dict[str, Any]:
        code = self.code

        url = self.url


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "code": code,
            "url": url,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        code = d.pop("code")

        url = d.pop("url")

        coupon_pool_tokens = cls(
            code=code,
            url=url,
        )

        return coupon_pool_tokens

