from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.provider_stream_role_role import ProviderStreamRoleRole
from ..models.provider_stream_role_stream import ProviderStreamRoleStream






T = TypeVar("T", bound="ProviderStreamRole")



@_attrs_define
class ProviderStreamRole:
    """ 
        Attributes:
            active (bool):
            held (bool):
            percent (int):
            role (ProviderStreamRoleRole):
            stream (ProviderStreamRoleStream):
     """

    active: bool
    held: bool
    percent: int
    role: ProviderStreamRoleRole
    stream: ProviderStreamRoleStream





    def to_dict(self) -> dict[str, Any]:
        active = self.active

        held = self.held

        percent = self.percent

        role = self.role.value

        stream = self.stream.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "active": active,
            "held": held,
            "percent": percent,
            "role": role,
            "stream": stream,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        active = d.pop("active")

        held = d.pop("held")

        percent = d.pop("percent")

        role = ProviderStreamRoleRole(d.pop("role"))




        stream = ProviderStreamRoleStream(d.pop("stream"))




        provider_stream_role = cls(
            active=active,
            held=held,
            percent=percent,
            role=role,
            stream=stream,
        )

        return provider_stream_role

