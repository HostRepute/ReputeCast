from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_route_source import ComposerRouteSource
from typing import cast

if TYPE_CHECKING:
  from ..models.route_share import RouteShare





T = TypeVar("T", bound="ComposerRoute")



@_attrs_define
class ComposerRoute:
    """ 
        Attributes:
            shares (list[RouteShare]):
            source (ComposerRouteSource):
     """

    shares: list[RouteShare]
    source: ComposerRouteSource





    def to_dict(self) -> dict[str, Any]:
        from ..models.route_share import RouteShare # noqa: PLC0415
        shares = []
        for shares_item_data in self.shares:
            shares_item = shares_item_data.to_dict()
            shares.append(shares_item)



        source = self.source.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "shares": shares,
            "source": source,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.route_share import RouteShare # noqa: PLC0415
        d = dict(src_dict)
        shares = []
        _shares = d.pop("shares")
        for shares_item_data in (_shares):
            shares_item = RouteShare.from_dict(shares_item_data)



            shares.append(shares_item)


        source = ComposerRouteSource(d.pop("source"))




        composer_route = cls(
            shares=shares,
            source=source,
        )

        return composer_route

