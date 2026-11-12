from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_upsert_input_consent import ContactUpsertInputConsent
from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_upsert_input_fields import ContactUpsertInputFields





T = TypeVar("T", bound="ContactUpsertInput")



@_attrs_define
class ContactUpsertInput:
    """ 
        Attributes:
            email (str):  Example: rob@example.net.
            company (str | Unset):
            consent (ContactUpsertInputConsent | Unset):
            fields (ContactUpsertInputFields | Unset):
            first_name (str | Unset):
            language (str | Unset):
            last_name (str | Unset):
            lists (list[str] | Unset):
            tags (list[str] | Unset):
     """

    email: str
    company: str | Unset = UNSET
    consent: ContactUpsertInputConsent | Unset = UNSET
    fields: ContactUpsertInputFields | Unset = UNSET
    first_name: str | Unset = UNSET
    language: str | Unset = UNSET
    last_name: str | Unset = UNSET
    lists: list[str] | Unset = UNSET
    tags: list[str] | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_upsert_input_fields import ContactUpsertInputFields # noqa: PLC0415
        email = self.email

        company = self.company

        consent: str | Unset = UNSET
        if not isinstance(self.consent, Unset):
            consent = self.consent.value


        fields: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fields, Unset):
            fields = self.fields.to_dict()

        first_name = self.first_name

        language = self.language

        last_name = self.last_name

        lists: list[str] | Unset = UNSET
        if not isinstance(self.lists, Unset):
            lists = self.lists



        tags: list[str] | Unset = UNSET
        if not isinstance(self.tags, Unset):
            tags = self.tags




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
        })
        if company is not UNSET:
            field_dict["company"] = company
        if consent is not UNSET:
            field_dict["consent"] = consent
        if fields is not UNSET:
            field_dict["fields"] = fields
        if first_name is not UNSET:
            field_dict["firstName"] = first_name
        if language is not UNSET:
            field_dict["language"] = language
        if last_name is not UNSET:
            field_dict["lastName"] = last_name
        if lists is not UNSET:
            field_dict["lists"] = lists
        if tags is not UNSET:
            field_dict["tags"] = tags

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_upsert_input_fields import ContactUpsertInputFields # noqa: PLC0415
        d = dict(src_dict)
        email = d.pop("email")

        company = d.pop("company", UNSET)

        _consent = d.pop("consent", UNSET)
        consent: ContactUpsertInputConsent | Unset
        if isinstance(_consent,  Unset):
            consent = UNSET
        else:
            consent = ContactUpsertInputConsent(_consent)




        _fields = d.pop("fields", UNSET)
        fields: ContactUpsertInputFields | Unset
        if isinstance(_fields,  Unset):
            fields = UNSET
        else:
            fields = ContactUpsertInputFields.from_dict(_fields)




        first_name = d.pop("firstName", UNSET)

        language = d.pop("language", UNSET)

        last_name = d.pop("lastName", UNSET)

        lists = cast(list[str], d.pop("lists", UNSET))


        tags = cast(list[str], d.pop("tags", UNSET))


        contact_upsert_input = cls(
            email=email,
            company=company,
            consent=consent,
            fields=fields,
            first_name=first_name,
            language=language,
            last_name=last_name,
            lists=lists,
            tags=tags,
        )

        return contact_upsert_input

