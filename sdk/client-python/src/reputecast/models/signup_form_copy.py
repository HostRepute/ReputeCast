from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.signup_form_copy_language_mode import SignupFormCopyLanguageMode
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.signup_form_copy_field import SignupFormCopyField
  from ..models.signup_form_copy_language import SignupFormCopyLanguage





T = TypeVar("T", bound="SignupFormCopy")



@_attrs_define
class SignupFormCopy:
    """ 
        Attributes:
            consent_changed_by (str):
            consent_version (int):
            copy_language (str): The language the form is written in
            fallback (str): The language the form shows where one isn't Live
            fields (list[SignupFormCopyField]):
            fixed_language (str):
            form_id (str):
            language_mode (SignupFormCopyLanguageMode):
            languages (list[SignupFormCopyLanguage]):
            consent_changed_at (datetime.datetime | Unset):
     """

    consent_changed_by: str
    consent_version: int
    copy_language: str
    fallback: str
    fields: list[SignupFormCopyField]
    fixed_language: str
    form_id: str
    language_mode: SignupFormCopyLanguageMode
    languages: list[SignupFormCopyLanguage]
    consent_changed_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.signup_form_copy_field import SignupFormCopyField # noqa: PLC0415
        from ..models.signup_form_copy_language import SignupFormCopyLanguage # noqa: PLC0415
        consent_changed_by = self.consent_changed_by

        consent_version = self.consent_version

        copy_language = self.copy_language

        fallback = self.fallback

        fields = []
        for fields_item_data in self.fields:
            fields_item = fields_item_data.to_dict()
            fields.append(fields_item)



        fixed_language = self.fixed_language

        form_id = self.form_id

        language_mode = self.language_mode.value

        languages = []
        for languages_item_data in self.languages:
            languages_item = languages_item_data.to_dict()
            languages.append(languages_item)



        consent_changed_at: str | Unset = UNSET
        if not isinstance(self.consent_changed_at, Unset):
            consent_changed_at = self.consent_changed_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "consentChangedBy": consent_changed_by,
            "consentVersion": consent_version,
            "copyLanguage": copy_language,
            "fallback": fallback,
            "fields": fields,
            "fixedLanguage": fixed_language,
            "formId": form_id,
            "languageMode": language_mode,
            "languages": languages,
        })
        if consent_changed_at is not UNSET:
            field_dict["consentChangedAt"] = consent_changed_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.signup_form_copy_field import SignupFormCopyField # noqa: PLC0415
        from ..models.signup_form_copy_language import SignupFormCopyLanguage # noqa: PLC0415
        d = dict(src_dict)
        consent_changed_by = d.pop("consentChangedBy")

        consent_version = d.pop("consentVersion")

        copy_language = d.pop("copyLanguage")

        fallback = d.pop("fallback")

        fields = []
        _fields = d.pop("fields")
        for fields_item_data in (_fields):
            fields_item = SignupFormCopyField.from_dict(fields_item_data)



            fields.append(fields_item)


        fixed_language = d.pop("fixedLanguage")

        form_id = d.pop("formId")

        language_mode = SignupFormCopyLanguageMode(d.pop("languageMode"))




        languages = []
        _languages = d.pop("languages")
        for languages_item_data in (_languages):
            languages_item = SignupFormCopyLanguage.from_dict(languages_item_data)



            languages.append(languages_item)


        _consent_changed_at = d.pop("consentChangedAt", UNSET)
        consent_changed_at: datetime.datetime | Unset
        if isinstance(_consent_changed_at,  Unset):
            consent_changed_at = UNSET
        else:
            consent_changed_at = datetime.datetime.fromisoformat(_consent_changed_at)




        signup_form_copy = cls(
            consent_changed_by=consent_changed_by,
            consent_version=consent_version,
            copy_language=copy_language,
            fallback=fallback,
            fields=fields,
            fixed_language=fixed_language,
            form_id=form_id,
            language_mode=language_mode,
            languages=languages,
            consent_changed_at=consent_changed_at,
        )

        return signup_form_copy

