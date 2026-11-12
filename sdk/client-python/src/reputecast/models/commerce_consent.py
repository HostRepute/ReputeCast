from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset






T = TypeVar("T", bound="CommerceConsent")



@_attrs_define
class CommerceConsent:
    """ 
        Attributes:
            list_created (bool):
            list_members (int):
            write_back (bool):
            write_back_supported (bool):
            write_granted (bool):
            list_id (str | Unset):
            list_name (str | Unset):
     """

    list_created: bool
    list_members: int
    write_back: bool
    write_back_supported: bool
    write_granted: bool
    list_id: str | Unset = UNSET
    list_name: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        list_created = self.list_created

        list_members = self.list_members

        write_back = self.write_back

        write_back_supported = self.write_back_supported

        write_granted = self.write_granted

        list_id = self.list_id

        list_name = self.list_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "listCreated": list_created,
            "listMembers": list_members,
            "writeBack": write_back,
            "writeBackSupported": write_back_supported,
            "writeGranted": write_granted,
        })
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if list_name is not UNSET:
            field_dict["listName"] = list_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        list_created = d.pop("listCreated")

        list_members = d.pop("listMembers")

        write_back = d.pop("writeBack")

        write_back_supported = d.pop("writeBackSupported")

        write_granted = d.pop("writeGranted")

        list_id = d.pop("listId", UNSET)

        list_name = d.pop("listName", UNSET)

        commerce_consent = cls(
            list_created=list_created,
            list_members=list_members,
            write_back=write_back,
            write_back_supported=write_back_supported,
            write_granted=write_granted,
            list_id=list_id,
            list_name=list_name,
        )

        return commerce_consent

