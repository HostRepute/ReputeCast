from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.survey_question import SurveyQuestion
  from ..models.survey_settings import SurveySettings





T = TypeVar("T", bound="SurveyPagePreviewInput")



@_attrs_define
class SurveyPagePreviewInput:
    """ 
        Attributes:
            name (str):
            questions (list[SurveyQuestion]):
            settings (SurveySettings):
            description (str | Unset):
            slug (str | Unset):
     """

    name: str
    questions: list[SurveyQuestion]
    settings: SurveySettings
    description: str | Unset = UNSET
    slug: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_question import SurveyQuestion # noqa: PLC0415
        from ..models.survey_settings import SurveySettings # noqa: PLC0415
        name = self.name

        questions = []
        for questions_item_data in self.questions:
            questions_item = questions_item_data.to_dict()
            questions.append(questions_item)



        settings = self.settings.to_dict()

        description = self.description

        slug = self.slug


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "name": name,
            "questions": questions,
            "settings": settings,
        })
        if description is not UNSET:
            field_dict["description"] = description
        if slug is not UNSET:
            field_dict["slug"] = slug

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_question import SurveyQuestion # noqa: PLC0415
        from ..models.survey_settings import SurveySettings # noqa: PLC0415
        d = dict(src_dict)
        name = d.pop("name")

        questions = []
        _questions = d.pop("questions")
        for questions_item_data in (_questions):
            questions_item = SurveyQuestion.from_dict(questions_item_data)



            questions.append(questions_item)


        settings = SurveySettings.from_dict(d.pop("settings"))




        description = d.pop("description", UNSET)

        slug = d.pop("slug", UNSET)

        survey_page_preview_input = cls(
            name=name,
            questions=questions,
            settings=settings,
            description=description,
            slug=slug,
        )

        return survey_page_preview_input

