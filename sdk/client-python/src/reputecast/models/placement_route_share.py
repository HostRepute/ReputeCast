from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="PlacementRouteShare")



@_attrs_define
class PlacementRouteShare:
    """ 
        Attributes:
            matched (list[str]):
            provider_id (str):
            reasons (list[str]):
            seeds (list[str]):
            fallback (str | Unset):
     """

    matched: list[str]
    provider_id: str
    reasons: list[str]
    seeds: list[str]
    fallback: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        matched = self.matched



        provider_id = self.provider_id

        reasons = self.reasons



        seeds = self.seeds



        fallback = self.fallback


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "matched": matched,
            "providerId": provider_id,
            "reasons": reasons,
            "seeds": seeds,
        })
        if fallback is not UNSET:
            field_dict["fallback"] = fallback

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        matched = cast(list[str], d.pop("matched"))


        provider_id = d.pop("providerId")

        reasons = cast(list[str], d.pop("reasons"))


        seeds = cast(list[str], d.pop("seeds"))


        fallback = d.pop("fallback", UNSET)

        placement_route_share = cls(
            matched=matched,
            provider_id=provider_id,
            reasons=reasons,
            seeds=seeds,
            fallback=fallback,
        )

        return placement_route_share

