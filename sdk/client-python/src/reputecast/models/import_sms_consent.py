from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from typing import cast






T = TypeVar("T", bound="ImportSmsConsent")



@_attrs_define
class ImportSmsConsent:
    """ 
        Attributes:
            consent_mapped (bool):
            evidence_mapped (bool):
            missing (list[str]):
     """

    consent_mapped: bool
    evidence_mapped: bool
    missing: list[str]





    def to_dict(self) -> dict[str, Any]:
        consent_mapped = self.consent_mapped

        evidence_mapped = self.evidence_mapped

        missing = self.missing




        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consentMapped": consent_mapped,
            "evidenceMapped": evidence_mapped,
            "missing": missing,
        })

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        consent_mapped = d.pop("consentMapped")

        evidence_mapped = d.pop("evidenceMapped")

        missing = cast(list[str], d.pop("missing"))


        import_sms_consent = cls(
            consent_mapped=consent_mapped,
            evidence_mapped=evidence_mapped,
            missing=missing,
        )

        return import_sms_consent

