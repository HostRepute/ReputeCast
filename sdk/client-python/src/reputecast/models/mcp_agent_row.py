from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.mcp_agent_row_status import MCPAgentRowStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.mcp_capability_cell import MCPCapabilityCell





T = TypeVar("T", bound="MCPAgentRow")



@_attrs_define
class MCPAgentRow:
    """ 
        Attributes:
            cells (list[MCPCapabilityCell]):
            created_by (str):
            id (str):
            masked (str):
            name (str):
            permissions (list[str]):
            status (MCPAgentRowStatus):
     """

    cells: list[MCPCapabilityCell]
    created_by: str
    id: str
    masked: str
    name: str
    permissions: list[str]
    status: MCPAgentRowStatus





    def to_dict(self) -> dict[str, Any]:
        from ..models.mcp_capability_cell import MCPCapabilityCell # noqa: PLC0415
        cells = []
        for cells_item_data in self.cells:
            cells_item = cells_item_data.to_dict()
            cells.append(cells_item)



        created_by = self.created_by

        id = self.id

        masked = self.masked

        name = self.name

        permissions = self.permissions



        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "cells": cells,
            "createdBy": created_by,
            "id": id,
            "masked": masked,
            "name": name,
            "permissions": permissions,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.mcp_capability_cell import MCPCapabilityCell # noqa: PLC0415
        d = dict(src_dict)
        cells = []
        _cells = d.pop("cells")
        for cells_item_data in (_cells):
            cells_item = MCPCapabilityCell.from_dict(cells_item_data)



            cells.append(cells_item)


        created_by = d.pop("createdBy")

        id = d.pop("id")

        masked = d.pop("masked")

        name = d.pop("name")

        permissions = cast(list[str], d.pop("permissions"))


        status = MCPAgentRowStatus(d.pop("status"))




        mcp_agent_row = cls(
            cells=cells,
            created_by=created_by,
            id=id,
            masked=masked,
            name=name,
            permissions=permissions,
            status=status,
        )

        return mcp_agent_row

