from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="ProviderTenant")



@_attrs_define
class ProviderTenant:
    """ 
        Attributes:
            name (str):
            policy (str | Unset):
     """

    name: str
    policy: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        name = self.name

        policy = self.policy


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
        })
        if policy is not UNSET:
            field_dict["policy"] = policy

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        name = d.pop("name")

        policy = d.pop("policy", UNSET)

        provider_tenant = cls(
            name=name,
            policy=policy,
        )

        return provider_tenant

