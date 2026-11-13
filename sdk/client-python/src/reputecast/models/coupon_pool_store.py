from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.coupon_pool_store_platform import CouponPoolStorePlatform






T = TypeVar("T", bound="CouponPoolStore")



@_attrs_define
class CouponPoolStore:
    """ 
        Attributes:
            connected (bool):
            domain (str):
            id (str):
            name (str):
            platform (CouponPoolStorePlatform):
     """

    connected: bool
    domain: str
    id: str
    name: str
    platform: CouponPoolStorePlatform





    def to_dict(self) -> dict[str, Any]:
        connected = self.connected

        domain = self.domain

        id = self.id

        name = self.name

        platform = self.platform.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "connected": connected,
            "domain": domain,
            "id": id,
            "name": name,
            "platform": platform,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        connected = d.pop("connected")

        domain = d.pop("domain")

        id = d.pop("id")

        name = d.pop("name")

        platform = CouponPoolStorePlatform(d.pop("platform"))




        coupon_pool_store = cls(
            connected=connected,
            domain=domain,
            id=id,
            name=name,
            platform=platform,
        )

        return coupon_pool_store

