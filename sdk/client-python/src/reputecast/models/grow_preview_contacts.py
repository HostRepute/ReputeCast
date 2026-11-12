from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.grow_preview_contacts_status import GrowPreviewContactsStatus
from typing import cast

if TYPE_CHECKING:
  from ..models.grow_preview_contact import GrowPreviewContact





T = TypeVar("T", bound="GrowPreviewContacts")



@_attrs_define
class GrowPreviewContacts:
    """ 
        Attributes:
            items (list[GrowPreviewContact]):
            list_name (str):
            program_id (str):
            status (GrowPreviewContactsStatus):
     """

    items: list[GrowPreviewContact]
    list_name: str
    program_id: str
    status: GrowPreviewContactsStatus





    def to_dict(self) -> dict[str, Any]:
        from ..models.grow_preview_contact import GrowPreviewContact # noqa: PLC0415
        items = []
        for items_item_data in self.items:
            items_item = items_item_data.to_dict()
            items.append(items_item)



        list_name = self.list_name

        program_id = self.program_id

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "items": items,
            "listName": list_name,
            "programId": program_id,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.grow_preview_contact import GrowPreviewContact # noqa: PLC0415
        d = dict(src_dict)
        items = []
        _items = d.pop("items")
        for items_item_data in (_items):
            items_item = GrowPreviewContact.from_dict(items_item_data)



            items.append(items_item)


        list_name = d.pop("listName")

        program_id = d.pop("programId")

        status = GrowPreviewContactsStatus(d.pop("status"))




        grow_preview_contacts = cls(
            items=items,
            list_name=list_name,
            program_id=program_id,
            status=status,
        )

        return grow_preview_contacts

