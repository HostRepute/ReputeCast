from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast

if TYPE_CHECKING:
  from ..models.contact_list_join import ContactListJoin





T = TypeVar("T", bound="ContactUpsertResult")



@_attrs_define
class ContactUpsertResult:
    """ 
        Attributes:
            confirmation_sent (bool):
            contact_id (str):
            created (bool):
            email (str):
            lists (list[ContactListJoin]):
            tags (list[str]):
     """

    confirmation_sent: bool
    contact_id: str
    created: bool
    email: str
    lists: list[ContactListJoin]
    tags: list[str]





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_list_join import ContactListJoin # noqa: PLC0415
        confirmation_sent = self.confirmation_sent

        contact_id = self.contact_id

        created = self.created

        email = self.email

        lists = []
        for lists_item_data in self.lists:
            lists_item = lists_item_data.to_dict()
            lists.append(lists_item)



        tags = self.tags




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "confirmationSent": confirmation_sent,
            "contactId": contact_id,
            "created": created,
            "email": email,
            "lists": lists,
            "tags": tags,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_list_join import ContactListJoin # noqa: PLC0415
        d = dict(src_dict)
        confirmation_sent = d.pop("confirmationSent")

        contact_id = d.pop("contactId")

        created = d.pop("created")

        email = d.pop("email")

        lists = []
        _lists = d.pop("lists")
        for lists_item_data in (_lists):
            lists_item = ContactListJoin.from_dict(lists_item_data)



            lists.append(lists_item)


        tags = cast(list[str], d.pop("tags"))


        contact_upsert_result = cls(
            confirmation_sent=confirmation_sent,
            contact_id=contact_id,
            created=created,
            email=email,
            lists=lists,
            tags=tags,
        )

        return contact_upsert_result

