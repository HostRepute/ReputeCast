from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.survey_status import SurveyStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.survey_link import SurveyLink
  from ..models.survey_problem import SurveyProblem
  from ..models.survey_question import SurveyQuestion
  from ..models.survey_settings import SurveySettings





T = TypeVar("T", bound="Survey")



@_attrs_define
class Survey:
    """ 
        Attributes:
            can_edit (bool):
            completed (int):
            created_at (datetime.datetime):
            created_by (str):
            description (str):
            edit_helpers (list[str]):
            id (str):
            linked (list[SurveyLink]):
            name (str):
            number (int):
            problems (list[SurveyProblem]):
            questions (list[SurveyQuestion]):
            settings (SurveySettings):
            slug (str):
            started (int):
            status (SurveyStatus):
            updated_at (datetime.datetime):
            url (str):
            variable (str):
            closed_at (datetime.datetime | Unset):
            closed_by (str | Unset):
            closes_at (datetime.datetime | Unset):
            published_at (datetime.datetime | Unset):
            published_by (str | Unset):
     """

    can_edit: bool
    completed: int
    created_at: datetime.datetime
    created_by: str
    description: str
    edit_helpers: list[str]
    id: str
    linked: list[SurveyLink]
    name: str
    number: int
    problems: list[SurveyProblem]
    questions: list[SurveyQuestion]
    settings: SurveySettings
    slug: str
    started: int
    status: SurveyStatus
    updated_at: datetime.datetime
    url: str
    variable: str
    closed_at: datetime.datetime | Unset = UNSET
    closed_by: str | Unset = UNSET
    closes_at: datetime.datetime | Unset = UNSET
    published_at: datetime.datetime | Unset = UNSET
    published_by: str | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.survey_link import SurveyLink # noqa: PLC0415
        from ..models.survey_problem import SurveyProblem # noqa: PLC0415
        from ..models.survey_question import SurveyQuestion # noqa: PLC0415
        from ..models.survey_settings import SurveySettings # noqa: PLC0415
        can_edit = self.can_edit

        completed = self.completed

        created_at = self.created_at.isoformat()

        created_by = self.created_by

        description = self.description

        edit_helpers = self.edit_helpers



        id = self.id

        linked = []
        for linked_item_data in self.linked:
            linked_item = linked_item_data.to_dict()
            linked.append(linked_item)



        name = self.name

        number = self.number

        problems = []
        for problems_item_data in self.problems:
            problems_item = problems_item_data.to_dict()
            problems.append(problems_item)



        questions = []
        for questions_item_data in self.questions:
            questions_item = questions_item_data.to_dict()
            questions.append(questions_item)



        settings = self.settings.to_dict()

        slug = self.slug

        started = self.started

        status = self.status.value

        updated_at = self.updated_at.isoformat()

        url = self.url

        variable = self.variable

        closed_at: str | Unset = UNSET
        if not isinstance(self.closed_at, Unset):
            closed_at = self.closed_at.isoformat()

        closed_by = self.closed_by

        closes_at: str | Unset = UNSET
        if not isinstance(self.closes_at, Unset):
            closes_at = self.closes_at.isoformat()

        published_at: str | Unset = UNSET
        if not isinstance(self.published_at, Unset):
            published_at = self.published_at.isoformat()

        published_by = self.published_by


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canEdit": can_edit,
            "completed": completed,
            "createdAt": created_at,
            "createdBy": created_by,
            "description": description,
            "editHelpers": edit_helpers,
            "id": id,
            "linked": linked,
            "name": name,
            "number": number,
            "problems": problems,
            "questions": questions,
            "settings": settings,
            "slug": slug,
            "started": started,
            "status": status,
            "updatedAt": updated_at,
            "url": url,
            "variable": variable,
        })
        if closed_at is not UNSET:
            field_dict["closedAt"] = closed_at
        if closed_by is not UNSET:
            field_dict["closedBy"] = closed_by
        if closes_at is not UNSET:
            field_dict["closesAt"] = closes_at
        if published_at is not UNSET:
            field_dict["publishedAt"] = published_at
        if published_by is not UNSET:
            field_dict["publishedBy"] = published_by

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.survey_link import SurveyLink # noqa: PLC0415
        from ..models.survey_problem import SurveyProblem # noqa: PLC0415
        from ..models.survey_question import SurveyQuestion # noqa: PLC0415
        from ..models.survey_settings import SurveySettings # noqa: PLC0415
        d = dict(src_dict)
        can_edit = d.pop("canEdit")

        completed = d.pop("completed")

        created_at = datetime.datetime.fromisoformat(d.pop("createdAt"))




        created_by = d.pop("createdBy")

        description = d.pop("description")

        edit_helpers = cast(list[str], d.pop("editHelpers"))


        id = d.pop("id")

        linked = []
        _linked = d.pop("linked")
        for linked_item_data in (_linked):
            linked_item = SurveyLink.from_dict(linked_item_data)



            linked.append(linked_item)


        name = d.pop("name")

        number = d.pop("number")

        problems = []
        _problems = d.pop("problems")
        for problems_item_data in (_problems):
            problems_item = SurveyProblem.from_dict(problems_item_data)



            problems.append(problems_item)


        questions = []
        _questions = d.pop("questions")
        for questions_item_data in (_questions):
            questions_item = SurveyQuestion.from_dict(questions_item_data)



            questions.append(questions_item)


        settings = SurveySettings.from_dict(d.pop("settings"))




        slug = d.pop("slug")

        started = d.pop("started")

        status = SurveyStatus(d.pop("status"))




        updated_at = datetime.datetime.fromisoformat(d.pop("updatedAt"))




        url = d.pop("url")

        variable = d.pop("variable")

        _closed_at = d.pop("closedAt", UNSET)
        closed_at: datetime.datetime | Unset
        if isinstance(_closed_at,  Unset):
            closed_at = UNSET
        else:
            closed_at = datetime.datetime.fromisoformat(_closed_at)




        closed_by = d.pop("closedBy", UNSET)

        _closes_at = d.pop("closesAt", UNSET)
        closes_at: datetime.datetime | Unset
        if isinstance(_closes_at,  Unset):
            closes_at = UNSET
        else:
            closes_at = datetime.datetime.fromisoformat(_closes_at)




        _published_at = d.pop("publishedAt", UNSET)
        published_at: datetime.datetime | Unset
        if isinstance(_published_at,  Unset):
            published_at = UNSET
        else:
            published_at = datetime.datetime.fromisoformat(_published_at)




        published_by = d.pop("publishedBy", UNSET)

        survey = cls(
            can_edit=can_edit,
            completed=completed,
            created_at=created_at,
            created_by=created_by,
            description=description,
            edit_helpers=edit_helpers,
            id=id,
            linked=linked,
            name=name,
            number=number,
            problems=problems,
            questions=questions,
            settings=settings,
            slug=slug,
            started=started,
            status=status,
            updated_at=updated_at,
            url=url,
            variable=variable,
            closed_at=closed_at,
            closed_by=closed_by,
            closes_at=closes_at,
            published_at=published_at,
            published_by=published_by,
        )

        return survey

