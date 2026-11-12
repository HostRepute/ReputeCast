from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.mcp_capability_cell_level import MCPCapabilityCellLevel






T = TypeVar("T", bound="MCPCapabilityCell")



@_attrs_define
class MCPCapabilityCell:
    """ 
        Attributes:
            capability (str):
            label (str):  Example: Pause only · never start.
            level (MCPCapabilityCellLevel):
     """

    capability: str
    label: str
    level: MCPCapabilityCellLevel





    def to_dict(self) -> dict[str, Any]:
        capability = self.capability

        label = self.label

        level = self.level.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "capability": capability,
            "label": label,
            "level": level,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        capability = d.pop("capability")

        label = d.pop("label")

        level = MCPCapabilityCellLevel(d.pop("level"))




        mcp_capability_cell = cls(
            capability=capability,
            label=label,
            level=level,
        )

        return mcp_capability_cell

