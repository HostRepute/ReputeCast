from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.composer_preview_contact import ComposerPreviewContact
  from ..models.composer_token_value import ComposerTokenValue
  from ..models.liquid_problem import LiquidProblem





T = TypeVar("T", bound="ComposerPreview")



@_attrs_define
class ComposerPreview:
    """ 
        Attributes:
            from_email (str):
            from_name (str):
            html (str):
            language (str):
            language_reason (str):
            preheader (str):
            problems (list[LiquidProblem]):
            rendered_at (datetime.datetime):
            subject (str):
            text (str):
            tokens (list[ComposerTokenValue]):
            contact (ComposerPreviewContact | Unset):
            counted (int | Unset):
            counted_lists_only (bool | Unset):
            missing_first_name (int | Unset):
     """

    from_email: str
    from_name: str
    html: str
    language: str
    language_reason: str
    preheader: str
    problems: list[LiquidProblem]
    rendered_at: datetime.datetime
    subject: str
    text: str
    tokens: list[ComposerTokenValue]
    contact: ComposerPreviewContact | Unset = UNSET
    counted: int | Unset = UNSET
    counted_lists_only: bool | Unset = UNSET
    missing_first_name: int | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.composer_preview_contact import ComposerPreviewContact # noqa: PLC0415
        from ..models.composer_token_value import ComposerTokenValue # noqa: PLC0415
        from ..models.liquid_problem import LiquidProblem # noqa: PLC0415
        from_email = self.from_email

        from_name = self.from_name

        html = self.html

        language = self.language

        language_reason = self.language_reason

        preheader = self.preheader

        problems = []
        for problems_item_data in self.problems:
            problems_item = problems_item_data.to_dict()
            problems.append(problems_item)



        rendered_at = self.rendered_at.isoformat()

        subject = self.subject

        text = self.text

        tokens = []
        for tokens_item_data in self.tokens:
            tokens_item = tokens_item_data.to_dict()
            tokens.append(tokens_item)



        contact: dict[str, Any] | Unset = UNSET
        if not isinstance(self.contact, Unset):
            contact = self.contact.to_dict()

        counted = self.counted

        counted_lists_only = self.counted_lists_only

        missing_first_name = self.missing_first_name


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "fromEmail": from_email,
            "fromName": from_name,
            "html": html,
            "language": language,
            "languageReason": language_reason,
            "preheader": preheader,
            "problems": problems,
            "renderedAt": rendered_at,
            "subject": subject,
            "text": text,
            "tokens": tokens,
        })
        if contact is not UNSET:
            field_dict["contact"] = contact
        if counted is not UNSET:
            field_dict["counted"] = counted
        if counted_lists_only is not UNSET:
            field_dict["countedListsOnly"] = counted_lists_only
        if missing_first_name is not UNSET:
            field_dict["missingFirstName"] = missing_first_name

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.composer_preview_contact import ComposerPreviewContact # noqa: PLC0415
        from ..models.composer_token_value import ComposerTokenValue # noqa: PLC0415
        from ..models.liquid_problem import LiquidProblem # noqa: PLC0415
        d = dict(src_dict)
        from_email = d.pop("fromEmail")

        from_name = d.pop("fromName")

        html = d.pop("html")

        language = d.pop("language")

        language_reason = d.pop("languageReason")

        preheader = d.pop("preheader")

        problems = []
        _problems = d.pop("problems")
        for problems_item_data in (_problems):
            problems_item = LiquidProblem.from_dict(problems_item_data)



            problems.append(problems_item)


        rendered_at = datetime.datetime.fromisoformat(d.pop("renderedAt"))




        subject = d.pop("subject")

        text = d.pop("text")

        tokens = []
        _tokens = d.pop("tokens")
        for tokens_item_data in (_tokens):
            tokens_item = ComposerTokenValue.from_dict(tokens_item_data)



            tokens.append(tokens_item)


        _contact = d.pop("contact", UNSET)
        contact: ComposerPreviewContact | Unset
        if isinstance(_contact,  Unset):
            contact = UNSET
        else:
            contact = ComposerPreviewContact.from_dict(_contact)




        counted = d.pop("counted", UNSET)

        counted_lists_only = d.pop("countedListsOnly", UNSET)

        missing_first_name = d.pop("missingFirstName", UNSET)

        composer_preview = cls(
            from_email=from_email,
            from_name=from_name,
            html=html,
            language=language,
            language_reason=language_reason,
            preheader=preheader,
            problems=problems,
            rendered_at=rendered_at,
            subject=subject,
            text=text,
            tokens=tokens,
            contact=contact,
            counted=counted,
            counted_lists_only=counted_lists_only,
            missing_first_name=missing_first_name,
        )

        return composer_preview

