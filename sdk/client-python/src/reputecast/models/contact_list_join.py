from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_list_join_status import ContactListJoinStatus






T = TypeVar("T", bound="ContactListJoin")



@_attrs_define
class ContactListJoin:
    """ 
        Attributes:
            id (str):
            name (str):
            status (ContactListJoinStatus):
     """

    id: str
    name: str
    status: ContactListJoinStatus





    def to_dict(self) -> dict[str, Any]:
        id = self.id

        name = self.name

        status = self.status.value


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "id": id,
            "name": name,
            "status": status,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        id = d.pop("id")

        name = d.pop("name")

        status = ContactListJoinStatus(d.pop("status"))




        contact_list_join = cls(
            id=id,
            name=name,
            status=status,
        )

        return contact_list_join

