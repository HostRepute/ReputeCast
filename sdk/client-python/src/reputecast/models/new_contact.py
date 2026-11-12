from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.new_contact_consent import NewContactConsent
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="NewContact")



@_attrs_define
class NewContact:
    """ 
        Attributes:
            consent (NewContactConsent):
            email (str):
            lists (list[str]): List ids
            first_name (str | Unset):
            language (str | Unset):
            last_name (str | Unset):
            tags (list[str] | Unset):
     """

    consent: NewContactConsent
    email: str
    lists: list[str]
    first_name: str | Unset = UNSET
    language: str | Unset = UNSET
    last_name: str | Unset = UNSET
    tags: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        consent = self.consent.value

        email = self.email

        lists = self.lists



        first_name = self.first_name

        language = self.language

        last_name = self.last_name

        tags: list[str] | Unset = UNSET
        if not isinstance(self.tags, Unset):
            tags = self.tags




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consent": consent,
            "email": email,
            "lists": lists,
        })
        if first_name is not UNSET:
            field_dict["firstName"] = first_name
        if language is not UNSET:
            field_dict["language"] = language
        if last_name is not UNSET:
            field_dict["lastName"] = last_name
        if tags is not UNSET:
            field_dict["tags"] = tags

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        consent = NewContactConsent(d.pop("consent"))




        email = d.pop("email")

        lists = cast(list[str], d.pop("lists"))


        first_name = d.pop("firstName", UNSET)

        language = d.pop("language", UNSET)

        last_name = d.pop("lastName", UNSET)

        tags = cast(list[str], d.pop("tags", UNSET))


        new_contact = cls(
            consent=consent,
            email=email,
            lists=lists,
            first_name=first_name,
            language=language,
            last_name=last_name,
            tags=tags,
        )

        return new_contact

