from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.composer_preview_request_sample import ComposerPreviewRequestSample
from ..models.composer_preview_request_variant import ComposerPreviewRequestVariant
from ..types import UNSET, Unset
from uuid import UUID






T = TypeVar("T", bound="ComposerPreviewRequest")



@_attrs_define
class ComposerPreviewRequest:
    """ 
        Attributes:
            contact_id (UUID | Unset):
            editing (str | Unset):
            html (str | Unset):
            language (str | Unset):
            preheader (str | Unset):
            sample (ComposerPreviewRequestSample | Unset):  Default: ComposerPreviewRequestSample.CONTACT.
            subject (str | Unset):
            variant (ComposerPreviewRequestVariant | Unset):
     """

    contact_id: UUID | Unset = UNSET
    editing: str | Unset = UNSET
    html: str | Unset = UNSET
    language: str | Unset = UNSET
    preheader: str | Unset = UNSET
    sample: ComposerPreviewRequestSample | Unset = ComposerPreviewRequestSample.CONTACT
    subject: str | Unset = UNSET
    variant: ComposerPreviewRequestVariant | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        contact_id: str | Unset = UNSET
        if not isinstance(self.contact_id, Unset):
            contact_id = str(self.contact_id)

        editing = self.editing

        html = self.html

        language = self.language

        preheader = self.preheader

        sample: str | Unset = UNSET
        if not isinstance(self.sample, Unset):
            sample = self.sample.value


        subject = self.subject

        variant: str | Unset = UNSET
        if not isinstance(self.variant, Unset):
            variant = self.variant.value



        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if contact_id is not UNSET:
            field_dict["contactId"] = contact_id
        if editing is not UNSET:
            field_dict["editing"] = editing
        if html is not UNSET:
            field_dict["html"] = html
        if language is not UNSET:
            field_dict["language"] = language
        if preheader is not UNSET:
            field_dict["preheader"] = preheader
        if sample is not UNSET:
            field_dict["sample"] = sample
        if subject is not UNSET:
            field_dict["subject"] = subject
        if variant is not UNSET:
            field_dict["variant"] = variant

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        _contact_id = d.pop("contactId", UNSET)
        contact_id: UUID | Unset
        if isinstance(_contact_id,  Unset):
            contact_id = UNSET
        else:
            contact_id = UUID(_contact_id)




        editing = d.pop("editing", UNSET)

        html = d.pop("html", UNSET)

        language = d.pop("language", UNSET)

        preheader = d.pop("preheader", UNSET)

        _sample = d.pop("sample", UNSET)
        sample: ComposerPreviewRequestSample | Unset
        if isinstance(_sample,  Unset):
            sample = UNSET
        else:
            sample = ComposerPreviewRequestSample(_sample)




        subject = d.pop("subject", UNSET)

        _variant = d.pop("variant", UNSET)
        variant: ComposerPreviewRequestVariant | Unset
        if isinstance(_variant,  Unset):
            variant = UNSET
        else:
            variant = ComposerPreviewRequestVariant(_variant)




        composer_preview_request = cls(
            contact_id=contact_id,
            editing=editing,
            html=html,
            language=language,
            preheader=preheader,
            sample=sample,
            subject=subject,
            variant=variant,
        )

        return composer_preview_request

