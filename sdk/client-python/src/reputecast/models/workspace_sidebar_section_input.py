from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset







T = TypeVar("T", bound="WorkspaceSidebarSectionInput")



@_attrs_define
class WorkspaceSidebarSectionInput:
    """ 
        Attributes:
            shown (bool): Show the section in the sidebar; false hides it
     """

    shown: bool





    def to_dict(self) -> dict[str, Any]:
        shown = self.shown


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "shown": shown,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        shown = d.pop("shown")

        workspace_sidebar_section_input = cls(
            shown=shown,
        )

        return workspace_sidebar_section_input

