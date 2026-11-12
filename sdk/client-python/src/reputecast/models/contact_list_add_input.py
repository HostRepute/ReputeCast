from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.contact_list_add_input_consent import ContactListAddInputConsent
from ..types import UNSET, Unset






T = TypeVar("T", bound="ContactListAddInput")



@_attrs_define
class ContactListAddInput:
    """ 
        Attributes:
            email (str):
            consent (ContactListAddInputConsent | Unset):
     """

    email: str
    consent: ContactListAddInputConsent | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        email = self.email

        consent: str | Unset = UNSET
        if not isinstance(self.consent, Unset):
            consent = self.consent.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "email": email,
        })
        if consent is not UNSET:
            field_dict["consent"] = consent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        email = d.pop("email")

        _consent = d.pop("consent", UNSET)
        consent: ContactListAddInputConsent | Unset
        if isinstance(_consent,  Unset):
            consent = UNSET
        else:
            consent = ContactListAddInputConsent(_consent)




        contact_list_add_input = cls(
            email=email,
            consent=consent,
        )

        return contact_list_add_input

