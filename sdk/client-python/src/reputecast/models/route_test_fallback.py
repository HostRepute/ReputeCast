from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.route_test_fallback_policy import RouteTestFallbackPolicy
from ..types import UNSET, Unset






T = TypeVar("T", bound="RouteTestFallback")



@_attrs_define
class RouteTestFallback:
    """ 
        Attributes:
            policy (RouteTestFallbackPolicy):
            provider (str | Unset):
            short (str | Unset):
     """

    policy: RouteTestFallbackPolicy
    provider: str | Unset = UNSET
    short: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        policy = self.policy.value

        provider = self.provider

        short = self.short


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "policy": policy,
        })
        if provider is not UNSET:
            field_dict["provider"] = provider
        if short is not UNSET:
            field_dict["short"] = short

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        policy = RouteTestFallbackPolicy(d.pop("policy"))




        provider = d.pop("provider", UNSET)

        short = d.pop("short", UNSET)

        route_test_fallback = cls(
            policy=policy,
            provider=provider,
            short=short,
        )

        return route_test_fallback

