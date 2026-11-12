from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.poll_results_kind import PollResultsKind
from ..models.poll_results_save import PollResultsSave
from ..models.poll_results_status import PollResultsStatus
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.poll_answer_count import PollAnswerCount
  from ..models.poll_automated_visits import PollAutomatedVisits
  from ..models.poll_breakdowns import PollBreakdowns
  from ..models.poll_nps import PollNPS
  from ..models.poll_over_time import PollOverTime





T = TypeVar("T", bound="PollResults")



@_attrs_define
class PollResults:
    """ 
        Attributes:
            answer_page (str):
            answers (int):
            automated (PollAutomatedVisits):
            block_id (str):
            breakdown (PollBreakdowns):
            campaign (str):
            can_close (bool):
            can_export_answers (bool):
            changed (int):
            close_helpers (list[str]):
            closed_as_scheduled (bool):
            comments (int):
            config (str):
            counts (list[PollAnswerCount]):
            delivered (int):
            export_helpers (list[str]):
            id (str):
            kind (PollResultsKind):
            live (bool):
            names (bool):
            non_contacts (int):
            nps_wording (bool):
            over_time (PollOverTime):
            percent (float):
            question (str):
            save (PollResultsSave):
            status (PollResultsStatus):
            tests (int):
            campaign_id (str | Unset):
            closed_at (datetime.datetime | Unset):
            closed_by (str | Unset):
            closes_at (datetime.datetime | Unset):
            field (str | Unset):
            high (str | Unset):
            low (str | Unset):
            nps (PollNPS | Unset):
            reopened_at (datetime.datetime | Unset):
            started_at (datetime.datetime | Unset):
     """

    answer_page: str
    answers: int
    automated: PollAutomatedVisits
    block_id: str
    breakdown: PollBreakdowns
    campaign: str
    can_close: bool
    can_export_answers: bool
    changed: int
    close_helpers: list[str]
    closed_as_scheduled: bool
    comments: int
    config: str
    counts: list[PollAnswerCount]
    delivered: int
    export_helpers: list[str]
    id: str
    kind: PollResultsKind
    live: bool
    names: bool
    non_contacts: int
    nps_wording: bool
    over_time: PollOverTime
    percent: float
    question: str
    save: PollResultsSave
    status: PollResultsStatus
    tests: int
    campaign_id: str | Unset = UNSET
    closed_at: datetime.datetime | Unset = UNSET
    closed_by: str | Unset = UNSET
    closes_at: datetime.datetime | Unset = UNSET
    field: str | Unset = UNSET
    high: str | Unset = UNSET
    low: str | Unset = UNSET
    nps: PollNPS | Unset = UNSET
    reopened_at: datetime.datetime | Unset = UNSET
    started_at: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.poll_answer_count import PollAnswerCount # noqa: PLC0415
        from ..models.poll_automated_visits import PollAutomatedVisits # noqa: PLC0415
        from ..models.poll_breakdowns import PollBreakdowns # noqa: PLC0415
        from ..models.poll_nps import PollNPS # noqa: PLC0415
        from ..models.poll_over_time import PollOverTime # noqa: PLC0415
        answer_page = self.answer_page

        answers = self.answers

        automated = self.automated.to_dict()

        block_id = self.block_id

        breakdown = self.breakdown.to_dict()

        campaign = self.campaign

        can_close = self.can_close

        can_export_answers = self.can_export_answers

        changed = self.changed

        close_helpers = self.close_helpers



        closed_as_scheduled = self.closed_as_scheduled

        comments = self.comments

        config = self.config

        counts = []
        for counts_item_data in self.counts:
            counts_item = counts_item_data.to_dict()
            counts.append(counts_item)



        delivered = self.delivered

        export_helpers = self.export_helpers



        id = self.id

        kind = self.kind.value

        live = self.live

        names = self.names

        non_contacts = self.non_contacts

        nps_wording = self.nps_wording

        over_time = self.over_time.to_dict()

        percent = self.percent

        question = self.question

        save = self.save.value

        status = self.status.value

        tests = self.tests

        campaign_id = self.campaign_id

        closed_at: str | Unset = UNSET
        if not isinstance(self.closed_at, Unset):
            closed_at = self.closed_at.isoformat()

        closed_by = self.closed_by

        closes_at: str | Unset = UNSET
        if not isinstance(self.closes_at, Unset):
            closes_at = self.closes_at.isoformat()

        field = self.field

        high = self.high

        low = self.low

        nps: dict[str, Any] | Unset = UNSET
        if not isinstance(self.nps, Unset):
            nps = self.nps.to_dict()

        reopened_at: str | Unset = UNSET
        if not isinstance(self.reopened_at, Unset):
            reopened_at = self.reopened_at.isoformat()

        started_at: str | Unset = UNSET
        if not isinstance(self.started_at, Unset):
            started_at = self.started_at.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "answerPage": answer_page,
            "answers": answers,
            "automated": automated,
            "blockId": block_id,
            "breakdown": breakdown,
            "campaign": campaign,
            "canClose": can_close,
            "canExportAnswers": can_export_answers,
            "changed": changed,
            "closeHelpers": close_helpers,
            "closedAsScheduled": closed_as_scheduled,
            "comments": comments,
            "config": config,
            "counts": counts,
            "delivered": delivered,
            "exportHelpers": export_helpers,
            "id": id,
            "kind": kind,
            "live": live,
            "names": names,
            "nonContacts": non_contacts,
            "npsWording": nps_wording,
            "overTime": over_time,
            "percent": percent,
            "question": question,
            "save": save,
            "status": status,
            "tests": tests,
        })
        if campaign_id is not UNSET:
            field_dict["campaignId"] = campaign_id
        if closed_at is not UNSET:
            field_dict["closedAt"] = closed_at
        if closed_by is not UNSET:
            field_dict["closedBy"] = closed_by
        if closes_at is not UNSET:
            field_dict["closesAt"] = closes_at
        if field is not UNSET:
            field_dict["field"] = field
        if high is not UNSET:
            field_dict["high"] = high
        if low is not UNSET:
            field_dict["low"] = low
        if nps is not UNSET:
            field_dict["nps"] = nps
        if reopened_at is not UNSET:
            field_dict["reopenedAt"] = reopened_at
        if started_at is not UNSET:
            field_dict["startedAt"] = started_at

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.poll_answer_count import PollAnswerCount # noqa: PLC0415
        from ..models.poll_automated_visits import PollAutomatedVisits # noqa: PLC0415
        from ..models.poll_breakdowns import PollBreakdowns # noqa: PLC0415
        from ..models.poll_nps import PollNPS # noqa: PLC0415
        from ..models.poll_over_time import PollOverTime # noqa: PLC0415
        d = dict(src_dict)
        answer_page = d.pop("answerPage")

        answers = d.pop("answers")

        automated = PollAutomatedVisits.from_dict(d.pop("automated"))




        block_id = d.pop("blockId")

        breakdown = PollBreakdowns.from_dict(d.pop("breakdown"))




        campaign = d.pop("campaign")

        can_close = d.pop("canClose")

        can_export_answers = d.pop("canExportAnswers")

        changed = d.pop("changed")

        close_helpers = cast(list[str], d.pop("closeHelpers"))


        closed_as_scheduled = d.pop("closedAsScheduled")

        comments = d.pop("comments")

        config = d.pop("config")

        counts = []
        _counts = d.pop("counts")
        for counts_item_data in (_counts):
            counts_item = PollAnswerCount.from_dict(counts_item_data)



            counts.append(counts_item)


        delivered = d.pop("delivered")

        export_helpers = cast(list[str], d.pop("exportHelpers"))


        id = d.pop("id")

        kind = PollResultsKind(d.pop("kind"))




        live = d.pop("live")

        names = d.pop("names")

        non_contacts = d.pop("nonContacts")

        nps_wording = d.pop("npsWording")

        over_time = PollOverTime.from_dict(d.pop("overTime"))




        percent = d.pop("percent")

        question = d.pop("question")

        save = PollResultsSave(d.pop("save"))




        status = PollResultsStatus(d.pop("status"))




        tests = d.pop("tests")

        campaign_id = d.pop("campaignId", UNSET)

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




        field = d.pop("field", UNSET)

        high = d.pop("high", UNSET)

        low = d.pop("low", UNSET)

        _nps = d.pop("nps", UNSET)
        nps: PollNPS | Unset
        if isinstance(_nps,  Unset):
            nps = UNSET
        else:
            nps = PollNPS.from_dict(_nps)




        _reopened_at = d.pop("reopenedAt", UNSET)
        reopened_at: datetime.datetime | Unset
        if isinstance(_reopened_at,  Unset):
            reopened_at = UNSET
        else:
            reopened_at = datetime.datetime.fromisoformat(_reopened_at)




        _started_at = d.pop("startedAt", UNSET)
        started_at: datetime.datetime | Unset
        if isinstance(_started_at,  Unset):
            started_at = UNSET
        else:
            started_at = datetime.datetime.fromisoformat(_started_at)




        poll_results = cls(
            answer_page=answer_page,
            answers=answers,
            automated=automated,
            block_id=block_id,
            breakdown=breakdown,
            campaign=campaign,
            can_close=can_close,
            can_export_answers=can_export_answers,
            changed=changed,
            close_helpers=close_helpers,
            closed_as_scheduled=closed_as_scheduled,
            comments=comments,
            config=config,
            counts=counts,
            delivered=delivered,
            export_helpers=export_helpers,
            id=id,
            kind=kind,
            live=live,
            names=names,
            non_contacts=non_contacts,
            nps_wording=nps_wording,
            over_time=over_time,
            percent=percent,
            question=question,
            save=save,
            status=status,
            tests=tests,
            campaign_id=campaign_id,
            closed_at=closed_at,
            closed_by=closed_by,
            closes_at=closes_at,
            field=field,
            high=high,
            low=low,
            nps=nps,
            reopened_at=reopened_at,
            started_at=started_at,
        )

        return poll_results

