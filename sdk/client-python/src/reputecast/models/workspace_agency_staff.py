from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.workspace_agency_staff_level import WorkspaceAgencyStaffLevel
from ..types import UNSET, Unset






T = TypeVar("T", bound="WorkspaceAgencyStaff")



@_attrs_define
class WorkspaceAgencyStaff:
    """ 
        Attributes:
            contact (bool):
            every_client (bool):
            level (WorkspaceAgencyStaffLevel):
            user_id (str):
            title (str | Unset):
     """

    contact: bool
    every_client: bool
    level: WorkspaceAgencyStaffLevel
    user_id: str
    title: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact = self.contact

        every_client = self.every_client

        level = self.level.value

        user_id = self.user_id

        title = self.title


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "contact": contact,
            "everyClient": every_client,
            "level": level,
            "userId": user_id,
        })
        if title is not UNSET:
            field_dict["title"] = title

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        contact = d.pop("contact")

        every_client = d.pop("everyClient")

        level = WorkspaceAgencyStaffLevel(d.pop("level"))




        user_id = d.pop("userId")

        title = d.pop("title", UNSET)

        workspace_agency_staff = cls(
            contact=contact,
            every_client=every_client,
            level=level,
            user_id=user_id,
            title=title,
        )

        return workspace_agency_staff

