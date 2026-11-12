from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="MCPToolInfo")



@_attrs_define
class MCPToolInfo:
    """ 
        Attributes:
            description (str):
            name (str):
            permission (str):
            writes (bool):
     """

    description: str
    name: str
    permission: str
    writes: bool





    def to_dict(self) -> dict[str, Any]:
        description = self.description

        name = self.name

        permission = self.permission

        writes = self.writes


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "description": description,
            "name": name,
            "permission": permission,
            "writes": writes,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        description = d.pop("description")

        name = d.pop("name")

        permission = d.pop("permission")

        writes = d.pop("writes")

        mcp_tool_info = cls(
            description=description,
            name=name,
            permission=permission,
            writes=writes,
        )

        return mcp_tool_info

