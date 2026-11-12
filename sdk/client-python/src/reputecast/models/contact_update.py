from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.contact_update_fields import ContactUpdateFields





T = TypeVar("T", bound="ContactUpdate")



@_attrs_define
class ContactUpdate:
    """ 
        Attributes:
            company (str | Unset):
            country (str | Unset):
            email (str | Unset):
            fields (ContactUpdateFields | Unset): Custom field key → value; null clears it
            first_name (str | Unset):
            language (str | Unset): A BCP 47 code such as es-MX; "Spanish" and "es_MX" are normalized. Empty clears it.
            last_name (str | Unset):
            tags (list[str] | Unset):
            timezone (str | Unset):
     """

    company: str | Unset = UNSET
    country: str | Unset = UNSET
    email: str | Unset = UNSET
    fields: ContactUpdateFields | Unset = UNSET
    first_name: str | Unset = UNSET
    language: str | Unset = UNSET
    last_name: str | Unset = UNSET
    tags: list[str] | Unset = UNSET
    timezone: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.contact_update_fields import ContactUpdateFields # noqa: PLC0415
        company = self.company

        country = self.country

        email = self.email

        fields: dict[str, Any] | Unset = UNSET
        if not isinstance(self.fields, Unset):
            fields = self.fields.to_dict()

        first_name = self.first_name

        language = self.language

        last_name = self.last_name

        tags: list[str] | Unset = UNSET
        if not isinstance(self.tags, Unset):
            tags = self.tags



        timezone = self.timezone


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if company is not UNSET:
            field_dict["company"] = company
        if country is not UNSET:
            field_dict["country"] = country
        if email is not UNSET:
            field_dict["email"] = email
        if fields is not UNSET:
            field_dict["fields"] = fields
        if first_name is not UNSET:
            field_dict["firstName"] = first_name
        if language is not UNSET:
            field_dict["language"] = language
        if last_name is not UNSET:
            field_dict["lastName"] = last_name
        if tags is not UNSET:
            field_dict["tags"] = tags
        if timezone is not UNSET:
            field_dict["timezone"] = timezone

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.contact_update_fields import ContactUpdateFields # noqa: PLC0415
        d = dict(src_dict)
        company = d.pop("company", UNSET)

        country = d.pop("country", UNSET)

        email = d.pop("email", UNSET)

        _fields = d.pop("fields", UNSET)
        fields: ContactUpdateFields | Unset
        if isinstance(_fields,  Unset):
            fields = UNSET
        else:
            fields = ContactUpdateFields.from_dict(_fields)




        first_name = d.pop("firstName", UNSET)

        language = d.pop("language", UNSET)

        last_name = d.pop("lastName", UNSET)

        tags = cast(list[str], d.pop("tags", UNSET))


        timezone = d.pop("timezone", UNSET)

        contact_update = cls(
            company=company,
            country=country,
            email=email,
            fields=fields,
            first_name=first_name,
            language=language,
            last_name=last_name,
            tags=tags,
            timezone=timezone,
        )

        return contact_update

