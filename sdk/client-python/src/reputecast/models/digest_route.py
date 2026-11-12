from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.digest_route_share import DigestRouteShare





T = TypeVar("T", bound="DigestRoute")



@_attrs_define
class DigestRoute:
    """ 
        Attributes:
            override (bool):
            shares (list[DigestRouteShare]):
     """

    override: bool
    shares: list[DigestRouteShare]





    def to_dict(self) -> dict[str, Any]:
        from ..models.digest_route_share import DigestRouteShare # noqa: PLC0415
        override = self.override

        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "override": override,
            "shares": shares,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.digest_route_share import DigestRouteShare # noqa: PLC0415
        d = dict(src_dict)
        override = d.pop("override")

        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = DigestRouteShare.from_dict(shares_item_data)



            shares.append(shares_item)


        digest_route = cls(
            override=override,
            shares=shares,
        )

        return digest_route

