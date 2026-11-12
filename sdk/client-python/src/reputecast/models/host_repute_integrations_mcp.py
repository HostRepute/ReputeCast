from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="HostReputeIntegrationsMCP")



@_attrs_define
class HostReputeIntegrationsMCP:
    """ 
        Attributes:
            agents (int):
     """

    agents: int





    def to_dict(self) -> dict[str, Any]:
        agents = self.agents


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "agents": agents,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        agents = d.pop("agents")

        host_repute_integrations_mcp = cls(
            agents=agents,
        )

        return host_repute_integrations_mcp

