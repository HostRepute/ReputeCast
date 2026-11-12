from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_test_send_sample import ComposerTestSendSample
from ..models.composer_test_send_variant import ComposerTestSendVariant
from ..types import UNSET, Unset
from typing import cast
from uuid import UUID






T = TypeVar("T", bound="ComposerTestSend")



@_attrs_define
class ComposerTestSend:
    """ 
        Attributes:
            to (list[str]):
            contact_id (UUID | Unset): Personalize as this contact
            each_language (bool | Unset):
            language (str | Unset):
            sample (ComposerTestSendSample | Unset): Personalize as Preview's reader without a first name
            variant (ComposerTestSendVariant | Unset):
     """

    to: list[str]
    contact_id: UUID | Unset = UNSET
    each_language: bool | Unset = UNSET
    language: str | Unset = UNSET
    sample: ComposerTestSendSample | Unset = UNSET
    variant: ComposerTestSendVariant | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        to = self.to



        contact_id: str | Unset = UNSET
        if not isinstance(self.contact_id, Unset):
            contact_id = str(self.contact_id)

        each_language = self.each_language

        language = self.language

        sample: str | Unset = UNSET
        if not isinstance(self.sample, Unset):
            sample = self.sample.value


        variant: str | Unset = UNSET
        if not isinstance(self.variant, Unset):
            variant = self.variant.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
            "to": to,
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if each_language is not UNSET:
            field_dict["eachLanguage"] = each_language
        if language is not UNSET:
            field_dict["language"] = language
        if sample is not UNSET:
            field_dict["sample"] = sample
        if variant is not UNSET:
            field_dict["variant"] = variant

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        to = cast(list[str], d.pop("to"))


        _contact_id = d.pop("contactId", UNSET)
        contact_id: UUID | Unset
        if isinstance(_contact_id,  Unset):
            contact_id = UNSET
        else:
            contact_id = UUID(_contact_id)




        each_language = d.pop("eachLanguage", UNSET)

        language = d.pop("language", UNSET)

        _sample = d.pop("sample", UNSET)
        sample: ComposerTestSendSample | Unset
        if isinstance(_sample,  Unset):
            sample = UNSET
        else:
            sample = ComposerTestSendSample(_sample)




        _variant = d.pop("variant", UNSET)
        variant: ComposerTestSendVariant | Unset
        if isinstance(_variant,  Unset):
            variant = UNSET
        else:
            variant = ComposerTestSendVariant(_variant)




        composer_test_send = cls(
            to=to,
            contact_id=contact_id,
            each_language=each_language,
            language=language,
            sample=sample,
            variant=variant,
        )

        return composer_test_send

