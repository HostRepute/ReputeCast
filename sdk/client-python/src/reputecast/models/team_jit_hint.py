from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="TeamJITHint")



@_attrs_define
class TeamJITHint:
    """ 
        Attributes:
            domain (str):
            preset (str):
            provider (str):
            role_name (str):
     """

    domain: str
    preset: str
    provider: str
    role_name: str





    def to_dict(self) -> dict[str, Any]:
        domain = self.domain

        preset = self.preset

        provider = self.provider

        role_name = self.role_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "domain": domain,
            "preset": preset,
            "provider": provider,
            "roleName": role_name,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        domain = d.pop("domain")

        preset = d.pop("preset")

        provider = d.pop("provider")

        role_name = d.pop("roleName")

        team_jit_hint = cls(
            domain=domain,
            preset=preset,
            provider=provider,
            role_name=role_name,
        )

        return team_jit_hint

