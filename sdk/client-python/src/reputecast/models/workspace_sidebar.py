from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.workspace_sidebar_hidden_sections_item import WorkspaceSidebarHiddenSectionsItem
from typing import cast






T = TypeVar("T", bound="WorkspaceSidebar")



@_attrs_define
class WorkspaceSidebar:
    """ 
        Attributes:
            hidden_sections (list[WorkspaceSidebarHiddenSectionsItem]): Sections hidden from the sidebar, in sidebar order
     """

    hidden_sections: list[WorkspaceSidebarHiddenSectionsItem]





    def to_dict(self) -> dict[str, Any]:
        hidden_sections = []
        for hidden_sections_item_data in self.hidden_sections:
            hidden_sections_item = hidden_sections_item_data.value
            hidden_sections.append(hidden_sections_item)




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "hiddenSections": hidden_sections,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        hidden_sections = []
        _hidden_sections = d.pop("hiddenSections")
        for hidden_sections_item_data in (_hidden_sections):
            hidden_sections_item = WorkspaceSidebarHiddenSectionsItem(hidden_sections_item_data)



            hidden_sections.append(hidden_sections_item)


        workspace_sidebar = cls(
            hidden_sections=hidden_sections,
        )

        return workspace_sidebar

