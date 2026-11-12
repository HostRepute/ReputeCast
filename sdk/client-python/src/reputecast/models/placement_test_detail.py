from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.placement_test_detail_content_kind import PlacementTestDetailContentKind
from ..models.placement_test_detail_delete_after import PlacementTestDetailDeleteAfter
from ..models.placement_test_detail_finish_reason import PlacementTestDetailFinishReason
from ..models.placement_test_detail_origin import PlacementTestDetailOrigin
from ..models.placement_test_detail_status import PlacementTestDetailStatus
from ..models.placement_test_detail_through import PlacementTestDetailThrough
from ..types import UNSET, Unset
from typing import cast
import datetime

if TYPE_CHECKING:
  from ..models.placement_copy_view import PlacementCopyView
  from ..models.placement_counts import PlacementCounts
  from ..models.placement_guidance import PlacementGuidance
  from ..models.placement_mailbox_result import PlacementMailboxResult
  from ..models.placement_provider_result import PlacementProviderResult
  from ..models.placement_result_seed import PlacementResultSeed





T = TypeVar("T", bound="PlacementTestDetail")



@_attrs_define
class PlacementTestDetail:
    """ 
        Attributes:
            can_run (bool):
            content_kind (PlacementTestDetailContentKind):
            content_name (str):
            content_note (str): draft, scheduled, sent, template or the transactional template's key
            copies (list[PlacementCopyView]):
            counts (PlacementCounts):
            delete_after (PlacementTestDetailDeleteAfter):
            from_email (str):
            from_name (str):
            gap_minutes (int):
            guidance (list[PlacementGuidance]):
            host_repute (bool):
            id (str):
            mailboxes (list[PlacementMailboxResult]):
            name (str):
            origin (PlacementTestDetailOrigin):
            personalization (str):
            providers (list[PlacementProviderResult]):
            rounds (int):
            run_helpers (list[str]):
            seeds (list[PlacementResultSeed]):
            started_at (datetime.datetime):
            started_by (str):
            status (PlacementTestDetailStatus):
            stream (str):
            subject (str):
            through (PlacementTestDetailThrough):
            conditional_shows (str | Unset):
            conditional_version (int | Unset):
            content_at (datetime.datetime | Unset): When the campaign is scheduled or was sent
            content_id (str | Unset):
            content_key (str | Unset):
            content_version (int | Unset):
            copies_deleted_at (datetime.datetime | Unset):
            copies_deleted_by (str | Unset):
            delete_at (datetime.datetime | Unset):
            fallback_greeting (str | Unset):
            finish_reason (PlacementTestDetailFinishReason | Unset):
            finished_at (datetime.datetime | Unset):
            language (str | Unset):
            last_sent_at (datetime.datetime | Unset):
            late_until (datetime.datetime | Unset): Late copies found until then update the result
            started_by_color (str | Unset):
            started_by_id (str | Unset):
            stopped_at (datetime.datetime | Unset):
            stopped_by (str | Unset):
            variant (str | Unset):
            wait_until (datetime.datetime | Unset): Copies missing then are Not arrived
     """

    can_run: bool
    content_kind: PlacementTestDetailContentKind
    content_name: str
    content_note: str
    copies: list[PlacementCopyView]
    counts: PlacementCounts
    delete_after: PlacementTestDetailDeleteAfter
    from_email: str
    from_name: str
    gap_minutes: int
    guidance: list[PlacementGuidance]
    host_repute: bool
    id: str
    mailboxes: list[PlacementMailboxResult]
    name: str
    origin: PlacementTestDetailOrigin
    personalization: str
    providers: list[PlacementProviderResult]
    rounds: int
    run_helpers: list[str]
    seeds: list[PlacementResultSeed]
    started_at: datetime.datetime
    started_by: str
    status: PlacementTestDetailStatus
    stream: str
    subject: str
    through: PlacementTestDetailThrough
    conditional_shows: str | Unset = UNSET
    conditional_version: int | Unset = UNSET
    content_at: datetime.datetime | Unset = UNSET
    content_id: str | Unset = UNSET
    content_key: str | Unset = UNSET
    content_version: int | Unset = UNSET
    copies_deleted_at: datetime.datetime | Unset = UNSET
    copies_deleted_by: str | Unset = UNSET
    delete_at: datetime.datetime | Unset = UNSET
    fallback_greeting: str | Unset = UNSET
    finish_reason: PlacementTestDetailFinishReason | Unset = UNSET
    finished_at: datetime.datetime | Unset = UNSET
    language: str | Unset = UNSET
    last_sent_at: datetime.datetime | Unset = UNSET
    late_until: datetime.datetime | Unset = UNSET
    started_by_color: str | Unset = UNSET
    started_by_id: str | Unset = UNSET
    stopped_at: datetime.datetime | Unset = UNSET
    stopped_by: str | Unset = UNSET
    variant: str | Unset = UNSET
    wait_until: datetime.datetime | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.placement_copy_view import PlacementCopyView # noqa: PLC0415
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        from ..models.placement_guidance import PlacementGuidance # noqa: PLC0415
        from ..models.placement_mailbox_result import PlacementMailboxResult # noqa: PLC0415
        from ..models.placement_provider_result import PlacementProviderResult # noqa: PLC0415
        from ..models.placement_result_seed import PlacementResultSeed # noqa: PLC0415
        can_run = self.can_run

        content_kind = self.content_kind.value

        content_name = self.content_name

        content_note = self.content_note

        copies = []
        for copies_item_data in self.copies:
            copies_item = copies_item_data.to_dict()
            copies.append(copies_item)



        counts = self.counts.to_dict()

        delete_after = self.delete_after.value

        from_email = self.from_email

        from_name = self.from_name

        gap_minutes = self.gap_minutes

        guidance = []
        for guidance_item_data in self.guidance:
            guidance_item = guidance_item_data.to_dict()
            guidance.append(guidance_item)



        host_repute = self.host_repute

        id = self.id

        mailboxes = []
        for mailboxes_item_data in self.mailboxes:
            mailboxes_item = mailboxes_item_data.to_dict()
            mailboxes.append(mailboxes_item)



        name = self.name

        origin = self.origin.value

        personalization = self.personalization

        providers = []
        for providers_item_data in self.providers:
            providers_item = providers_item_data.to_dict()
            providers.append(providers_item)



        rounds = self.rounds

        run_helpers = self.run_helpers



        seeds = []
        for seeds_item_data in self.seeds:
            seeds_item = seeds_item_data.to_dict()
            seeds.append(seeds_item)



        started_at = self.started_at.isoformat()

        started_by = self.started_by

        status = self.status.value

        stream = self.stream

        subject = self.subject

        through = self.through.value

        conditional_shows = self.conditional_shows

        conditional_version = self.conditional_version

        content_at: str | Unset = UNSET
        if not isinstance(self.content_at, Unset):
            content_at = self.content_at.isoformat()

        content_id = self.content_id

        content_key = self.content_key

        content_version = self.content_version

        copies_deleted_at: str | Unset = UNSET
        if not isinstance(self.copies_deleted_at, Unset):
            copies_deleted_at = self.copies_deleted_at.isoformat()

        copies_deleted_by = self.copies_deleted_by

        delete_at: str | Unset = UNSET
        if not isinstance(self.delete_at, Unset):
            delete_at = self.delete_at.isoformat()

        fallback_greeting = self.fallback_greeting

        finish_reason: str | Unset = UNSET
        if not isinstance(self.finish_reason, Unset):
            finish_reason = self.finish_reason.value


        finished_at: str | Unset = UNSET
        if not isinstance(self.finished_at, Unset):
            finished_at = self.finished_at.isoformat()

        language = self.language

        last_sent_at: str | Unset = UNSET
        if not isinstance(self.last_sent_at, Unset):
            last_sent_at = self.last_sent_at.isoformat()

        late_until: str | Unset = UNSET
        if not isinstance(self.late_until, Unset):
            late_until = self.late_until.isoformat()

        started_by_color = self.started_by_color

        started_by_id = self.started_by_id

        stopped_at: str | Unset = UNSET
        if not isinstance(self.stopped_at, Unset):
            stopped_at = self.stopped_at.isoformat()

        stopped_by = self.stopped_by

        variant = self.variant

        wait_until: str | Unset = UNSET
        if not isinstance(self.wait_until, Unset):
            wait_until = self.wait_until.isoformat()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "canRun": can_run,
            "contentKind": content_kind,
            "contentName": content_name,
            "contentNote": content_note,
            "copies": copies,
            "counts": counts,
            "deleteAfter": delete_after,
            "fromEmail": from_email,
            "fromName": from_name,
            "gapMinutes": gap_minutes,
            "guidance": guidance,
            "hostRepute": host_repute,
            "id": id,
            "mailboxes": mailboxes,
            "name": name,
            "origin": origin,
            "personalization": personalization,
            "providers": providers,
            "rounds": rounds,
            "runHelpers": run_helpers,
            "seeds": seeds,
            "startedAt": started_at,
            "startedBy": started_by,
            "status": status,
            "stream": stream,
            "subject": subject,
            "through": through,
        })
        if conditional_shows is not UNSET:
            field_dict["conditionalShows"] = conditional_shows
        if conditional_version is not UNSET:
            field_dict["conditionalVersion"] = conditional_version
        if content_at is not UNSET:
            field_dict["contentAt"] = content_at
        if content_id is not UNSET:
            field_dict["contentId"] = content_id
        if content_key is not UNSET:
            field_dict["contentKey"] = content_key
        if content_version is not UNSET:
            field_dict["contentVersion"] = content_version
        if copies_deleted_at is not UNSET:
            field_dict["copiesDeletedAt"] = copies_deleted_at
        if copies_deleted_by is not UNSET:
            field_dict["copiesDeletedBy"] = copies_deleted_by
        if delete_at is not UNSET:
            field_dict["deleteAt"] = delete_at
        if fallback_greeting is not UNSET:
            field_dict["fallbackGreeting"] = fallback_greeting
        if finish_reason is not UNSET:
            field_dict["finishReason"] = finish_reason
        if finished_at is not UNSET:
            field_dict["finishedAt"] = finished_at
        if language is not UNSET:
            field_dict["language"] = language
        if last_sent_at is not UNSET:
            field_dict["lastSentAt"] = last_sent_at
        if late_until is not UNSET:
            field_dict["lateUntil"] = late_until
        if started_by_color is not UNSET:
            field_dict["startedByColor"] = started_by_color
        if started_by_id is not UNSET:
            field_dict["startedById"] = started_by_id
        if stopped_at is not UNSET:
            field_dict["stoppedAt"] = stopped_at
        if stopped_by is not UNSET:
            field_dict["stoppedBy"] = stopped_by
        if variant is not UNSET:
            field_dict["variant"] = variant
        if wait_until is not UNSET:
            field_dict["waitUntil"] = wait_until

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.placement_copy_view import PlacementCopyView # noqa: PLC0415
        from ..models.placement_counts import PlacementCounts # noqa: PLC0415
        from ..models.placement_guidance import PlacementGuidance # noqa: PLC0415
        from ..models.placement_mailbox_result import PlacementMailboxResult # noqa: PLC0415
        from ..models.placement_provider_result import PlacementProviderResult # noqa: PLC0415
        from ..models.placement_result_seed import PlacementResultSeed # noqa: PLC0415
        d = dict(src_dict)
        can_run = d.pop("canRun")

        content_kind = PlacementTestDetailContentKind(d.pop("contentKind"))




        content_name = d.pop("contentName")

        content_note = d.pop("contentNote")

        copies = []
        _copies = d.pop("copies")
        for copies_item_data in (_copies):
            copies_item = PlacementCopyView.from_dict(copies_item_data)



            copies.append(copies_item)


        counts = PlacementCounts.from_dict(d.pop("counts"))




        delete_after = PlacementTestDetailDeleteAfter(d.pop("deleteAfter"))




        from_email = d.pop("fromEmail")

        from_name = d.pop("fromName")

        gap_minutes = d.pop("gapMinutes")

        guidance = []
        _guidance = d.pop("guidance")
        for guidance_item_data in (_guidance):
            guidance_item = PlacementGuidance.from_dict(guidance_item_data)



            guidance.append(guidance_item)


        host_repute = d.pop("hostRepute")

        id = d.pop("id")

        mailboxes = []
        _mailboxes = d.pop("mailboxes")
        for mailboxes_item_data in (_mailboxes):
            mailboxes_item = PlacementMailboxResult.from_dict(mailboxes_item_data)



            mailboxes.append(mailboxes_item)


        name = d.pop("name")

        origin = PlacementTestDetailOrigin(d.pop("origin"))




        personalization = d.pop("personalization")

        providers = []
        _providers = d.pop("providers")
        for providers_item_data in (_providers):
            providers_item = PlacementProviderResult.from_dict(providers_item_data)



            providers.append(providers_item)


        rounds = d.pop("rounds")

        run_helpers = cast(list[str], d.pop("runHelpers"))


        seeds = []
        _seeds = d.pop("seeds")
        for seeds_item_data in (_seeds):
            seeds_item = PlacementResultSeed.from_dict(seeds_item_data)



            seeds.append(seeds_item)


        started_at = datetime.datetime.fromisoformat(d.pop("startedAt"))




        started_by = d.pop("startedBy")

        status = PlacementTestDetailStatus(d.pop("status"))




        stream = d.pop("stream")

        subject = d.pop("subject")

        through = PlacementTestDetailThrough(d.pop("through"))




        conditional_shows = d.pop("conditionalShows", UNSET)

        conditional_version = d.pop("conditionalVersion", UNSET)

        _content_at = d.pop("contentAt", UNSET)
        content_at: datetime.datetime | Unset
        if isinstance(_content_at,  Unset):
            content_at = UNSET
        else:
            content_at = datetime.datetime.fromisoformat(_content_at)




        content_id = d.pop("contentId", UNSET)

        content_key = d.pop("contentKey", UNSET)

        content_version = d.pop("contentVersion", UNSET)

        _copies_deleted_at = d.pop("copiesDeletedAt", UNSET)
        copies_deleted_at: datetime.datetime | Unset
        if isinstance(_copies_deleted_at,  Unset):
            copies_deleted_at = UNSET
        else:
            copies_deleted_at = datetime.datetime.fromisoformat(_copies_deleted_at)




        copies_deleted_by = d.pop("copiesDeletedBy", UNSET)

        _delete_at = d.pop("deleteAt", UNSET)
        delete_at: datetime.datetime | Unset
        if isinstance(_delete_at,  Unset):
            delete_at = UNSET
        else:
            delete_at = datetime.datetime.fromisoformat(_delete_at)




        fallback_greeting = d.pop("fallbackGreeting", UNSET)

        _finish_reason = d.pop("finishReason", UNSET)
        finish_reason: PlacementTestDetailFinishReason | Unset
        if isinstance(_finish_reason,  Unset):
            finish_reason = UNSET
        else:
            finish_reason = PlacementTestDetailFinishReason(_finish_reason)




        _finished_at = d.pop("finishedAt", UNSET)
        finished_at: datetime.datetime | Unset
        if isinstance(_finished_at,  Unset):
            finished_at = UNSET
        else:
            finished_at = datetime.datetime.fromisoformat(_finished_at)




        language = d.pop("language", UNSET)

        _last_sent_at = d.pop("lastSentAt", UNSET)
        last_sent_at: datetime.datetime | Unset
        if isinstance(_last_sent_at,  Unset):
            last_sent_at = UNSET
        else:
            last_sent_at = datetime.datetime.fromisoformat(_last_sent_at)




        _late_until = d.pop("lateUntil", UNSET)
        late_until: datetime.datetime | Unset
        if isinstance(_late_until,  Unset):
            late_until = UNSET
        else:
            late_until = datetime.datetime.fromisoformat(_late_until)




        started_by_color = d.pop("startedByColor", UNSET)

        started_by_id = d.pop("startedById", UNSET)

        _stopped_at = d.pop("stoppedAt", UNSET)
        stopped_at: datetime.datetime | Unset
        if isinstance(_stopped_at,  Unset):
            stopped_at = UNSET
        else:
            stopped_at = datetime.datetime.fromisoformat(_stopped_at)




        stopped_by = d.pop("stoppedBy", UNSET)

        variant = d.pop("variant", UNSET)

        _wait_until = d.pop("waitUntil", UNSET)
        wait_until: datetime.datetime | Unset
        if isinstance(_wait_until,  Unset):
            wait_until = UNSET
        else:
            wait_until = datetime.datetime.fromisoformat(_wait_until)




        placement_test_detail = cls(
            can_run=can_run,
            content_kind=content_kind,
            content_name=content_name,
            content_note=content_note,
            copies=copies,
            counts=counts,
            delete_after=delete_after,
            from_email=from_email,
            from_name=from_name,
            gap_minutes=gap_minutes,
            guidance=guidance,
            host_repute=host_repute,
            id=id,
            mailboxes=mailboxes,
            name=name,
            origin=origin,
            personalization=personalization,
            providers=providers,
            rounds=rounds,
            run_helpers=run_helpers,
            seeds=seeds,
            started_at=started_at,
            started_by=started_by,
            status=status,
            stream=stream,
            subject=subject,
            through=through,
            conditional_shows=conditional_shows,
            conditional_version=conditional_version,
            content_at=content_at,
            content_id=content_id,
            content_key=content_key,
            content_version=content_version,
            copies_deleted_at=copies_deleted_at,
            copies_deleted_by=copies_deleted_by,
            delete_at=delete_at,
            fallback_greeting=fallback_greeting,
            finish_reason=finish_reason,
            finished_at=finished_at,
            language=language,
            last_sent_at=last_sent_at,
            late_until=late_until,
            started_by_color=started_by_color,
            started_by_id=started_by_id,
            stopped_at=stopped_at,
            stopped_by=stopped_by,
            variant=variant,
            wait_until=wait_until,
        )

        return placement_test_detail

