from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.grow_block_render_input_kind import GrowBlockRenderInputKind
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="GrowBlockRenderInput")



@_attrs_define
class GrowBlockRenderInput:
    """ 
        Attributes:
            kind (GrowBlockRenderInputKind):
            contact_id (UUID | Unset):
            milestone_id (UUID | Unset):
            settings (Any | Unset):
     """

    kind: GrowBlockRenderInputKind
    contact_id: UUID | Unset = UNSET
    milestone_id: UUID | Unset = UNSET
    settings: Any | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        contact_id: str | Unset = UNSET
        if not isinstance(self.contact_id, Unset):
            contact_id = str(self.contact_id)

        milestone_id: str | Unset = UNSET
        if not isinstance(self.milestone_id, Unset):
            milestone_id = str(self.milestone_id)

        settings = self.settings


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if milestone_id is not UNSET:
            field_dict["milestoneId"] = milestone_id
        if settings is not UNSET:
            field_dict["settings"] = settings

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = GrowBlockRenderInputKind(d.pop("kind"))




        _contact_id = d.pop("contactId", UNSET)
        contact_id: UUID | Unset
        if isinstance(_contact_id,  Unset):
            contact_id = UNSET
        else:
            contact_id = UUID(_contact_id)




        _milestone_id = d.pop("milestoneId", UNSET)
        milestone_id: UUID | Unset
        if isinstance(_milestone_id,  Unset):
            milestone_id = UNSET
        else:
            milestone_id = UUID(_milestone_id)




        settings = d.pop("settings", UNSET)

        grow_block_render_input = cls(
            kind=kind,
            contact_id=contact_id,
            milestone_id=milestone_id,
            settings=settings,
        )

        return grow_block_render_input

