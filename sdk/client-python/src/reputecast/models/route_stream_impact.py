from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_stream_impact_basis import RouteStreamImpactBasis
from ..models.route_stream_impact_stream import RouteStreamImpactStream
from ..types import UNSET, Unset






T = TypeVar("T", bound="RouteStreamImpact")



@_attrs_define
class RouteStreamImpact:
    """ 
        Attributes:
            addresses (int):
            basis (RouteStreamImpactBasis):
            moved (int):
            stream (RouteStreamImpactStream):
            from_ (str | Unset):
            to (str | Unset):
     """

    addresses: int
    basis: RouteStreamImpactBasis
    moved: int
    stream: RouteStreamImpactStream
    from_: str | Unset = UNSET
    to: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        addresses = self.addresses

        basis = self.basis.value

        moved = self.moved

        stream = self.stream.value

        from_ = self.from_

        to = self.to


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "addresses": addresses,
            "basis": basis,
            "moved": moved,
            "stream": stream,
        })
        if from_ is not UNSET:
            field_dict["from"] = from_
        if to is not UNSET:
            field_dict["to"] = to

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        addresses = d.pop("addresses")

        basis = RouteStreamImpactBasis(d.pop("basis"))




        moved = d.pop("moved")

        stream = RouteStreamImpactStream(d.pop("stream"))




        from_ = d.pop("from", UNSET)

        to = d.pop("to", UNSET)

        route_stream_impact = cls(
            addresses=addresses,
            basis=basis,
            moved=moved,
            stream=stream,
            from_=from_,
            to=to,
        )

        return route_stream_impact

