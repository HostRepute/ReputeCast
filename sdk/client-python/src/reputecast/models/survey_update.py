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





T = TypeVar("T", bound="SurveyUpdate")



@_attrs_define
class SurveyUpdate:
    """ 
        Attributes:
            description (str | Unset):
            name (str | Unset):
            questions (list[SurveyQuestion] | Unset):
            settings (SurveySettings | Unset):
            slug (str | Unset):
     """

    description: str | Unset = UNSET
    name: str | Unset = UNSET
    questions: list[SurveyQuestion] | Unset = UNSET
    settings: SurveySettings | Unset = UNSET
    slug: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_question import SurveyQuestion # noqa: PLC0415
        from ..models.survey_settings import SurveySettings # noqa: PLC0415
        description = self.description

        name = self.name

        questions: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.questions, Unset):
            questions = []
            for questions_item_data in self.questions:
                questions_item = questions_item_data.to_dict()
                questions.append(questions_item)



        settings: dict[str, Any] | Unset = UNSET
        if not isinstance(self.settings, Unset):
            settings = self.settings.to_dict()

        slug = self.slug


        field_dict: dict[str, Any] = {}

        field_dict.update({
        })
        if description is not UNSET:
            field_dict["description"] = description
        if name is not UNSET:
            field_dict["name"] = name
        if questions is not UNSET:
            field_dict["questions"] = questions
        if settings is not UNSET:
            field_dict["settings"] = settings
        if slug is not UNSET:
            field_dict["slug"] = slug

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_question import SurveyQuestion # noqa: PLC0415
        from ..models.survey_settings import SurveySettings # noqa: PLC0415
        d = dict(src_dict)
        description = d.pop("description", UNSET)

        name = d.pop("name", UNSET)

        _questions = d.pop("questions", UNSET)
        questions: list[SurveyQuestion] | Unset = UNSET
        if _questions is not UNSET:
            questions = []
            for questions_item_data in _questions:
                questions_item = SurveyQuestion.from_dict(questions_item_data)



                questions.append(questions_item)


        _settings = d.pop("settings", UNSET)
        settings: SurveySettings | Unset
        if isinstance(_settings,  Unset):
            settings = UNSET
        else:
            settings = SurveySettings.from_dict(_settings)




        slug = d.pop("slug", UNSET)

        survey_update = cls(
            description=description,
            name=name,
            questions=questions,
            settings=settings,
            slug=slug,
        )

        return survey_update

