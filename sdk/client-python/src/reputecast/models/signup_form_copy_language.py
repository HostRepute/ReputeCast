from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_copy_language_status import SignupFormCopyLanguageStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.public_copy_labels import PublicCopyLabels
  from ..models.signup_form_consent_evidence import SignupFormConsentEvidence
  from ..models.signup_form_copy_text import SignupFormCopyText





T = TypeVar("T", bound="SignupFormCopyLanguage")



@_attrs_define
class SignupFormCopyLanguage:
    """ 
        Attributes:
            by (str):
            code (str):  Example: es.
            consent_for (int):
            consent_review (bool): The form's own consent text changed since: Review the translation
            consent_version (int):
            copy (SignupFormCopyText):
            default_after_submit (str):
            is_default (bool):
            is_source (bool):
            labels (PublicCopyLabels):
            missing (int):
            name (str):  Example: Spanish.
            native (str):  Example: Español.
            status (SignupFormCopyLanguageStatus):
            total (int):
            written (int):
            at (datetime.datetime | Unset):
            evidence (SignupFormConsentEvidence | Unset):
     """

    by: str
    code: str
    consent_for: int
    consent_review: bool
    consent_version: int
    copy: SignupFormCopyText
    default_after_submit: str
    is_default: bool
    is_source: bool
    labels: PublicCopyLabels
    missing: int
    name: str
    native: str
    status: SignupFormCopyLanguageStatus
    total: int
    written: int
    at: datetime.datetime | Unset = UNSET
    evidence: SignupFormConsentEvidence | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.public_copy_labels import PublicCopyLabels # noqa: PLC0415
        from ..models.signup_form_consent_evidence import SignupFormConsentEvidence # noqa: PLC0415
        from ..models.signup_form_copy_text import SignupFormCopyText # noqa: PLC0415
        by = self.by

        code = self.code

        consent_for = self.consent_for

        consent_review = self.consent_review

        consent_version = self.consent_version

        copy = self.copy.to_dict()

        default_after_submit = self.default_after_submit

        is_default = self.is_default

        is_source = self.is_source

        labels = self.labels.to_dict()

        missing = self.missing

        name = self.name

        native = self.native

        status = self.status.value

        total = self.total

        written = self.written

        at: str | Unset = UNSET
        if not isinstance(self.at, Unset):
            at = self.at.isoformat()

        evidence: dict[str, Any] | Unset = UNSET
        if not isinstance(self.evidence, Unset):
            evidence = self.evidence.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "by": by,
            "code": code,
            "consentFor": consent_for,
            "consentReview": consent_review,
            "consentVersion": consent_version,
            "copy": copy,
            "defaultAfterSubmit": default_after_submit,
            "isDefault": is_default,
            "isSource": is_source,
            "labels": labels,
            "missing": missing,
            "name": name,
            "native": native,
            "status": status,
            "total": total,
            "written": written,
        })
        if at is not UNSET:
            field_dict["at"] = at
        if evidence is not UNSET:
            field_dict["evidence"] = evidence

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.public_copy_labels import PublicCopyLabels # noqa: PLC0415
        from ..models.signup_form_consent_evidence import SignupFormConsentEvidence # noqa: PLC0415
        from ..models.signup_form_copy_text import SignupFormCopyText # noqa: PLC0415
        d = dict(src_dict)
        by = d.pop("by")

        code = d.pop("code")

        consent_for = d.pop("consentFor")

        consent_review = d.pop("consentReview")

        consent_version = d.pop("consentVersion")

        copy = SignupFormCopyText.from_dict(d.pop("copy"))




        default_after_submit = d.pop("defaultAfterSubmit")

        is_default = d.pop("isDefault")

        is_source = d.pop("isSource")

        labels = PublicCopyLabels.from_dict(d.pop("labels"))




        missing = d.pop("missing")

        name = d.pop("name")

        native = d.pop("native")

        status = SignupFormCopyLanguageStatus(d.pop("status"))




        total = d.pop("total")

        written = d.pop("written")

        _at = d.pop("at", UNSET)
        at: datetime.datetime | Unset
        if isinstance(_at,  Unset):
            at = UNSET
        else:
            at = datetime.datetime.fromisoformat(_at)




        _evidence = d.pop("evidence", UNSET)
        evidence: SignupFormConsentEvidence | Unset
        if isinstance(_evidence,  Unset):
            evidence = UNSET
        else:
            evidence = SignupFormConsentEvidence.from_dict(_evidence)




        signup_form_copy_language = cls(
            by=by,
            code=code,
            consent_for=consent_for,
            consent_review=consent_review,
            consent_version=consent_version,
            copy=copy,
            default_after_submit=default_after_submit,
            is_default=is_default,
            is_source=is_source,
            labels=labels,
            missing=missing,
            name=name,
            native=native,
            status=status,
            total=total,
            written=written,
            at=at,
            evidence=evidence,
        )

        return signup_form_copy_language

