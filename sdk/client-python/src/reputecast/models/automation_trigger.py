from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..models.automation_trigger_answer_op import AutomationTriggerAnswerOp
from ..models.automation_trigger_direction import AutomationTriggerDirection
from ..models.automation_trigger_end_reasons_item import AutomationTriggerEndReasonsItem
from ..models.automation_trigger_kind import AutomationTriggerKind
from ..types import UNSET, Unset
from typing import cast






T = TypeVar("T", bound="AutomationTrigger")



@_attrs_define
class AutomationTrigger:
    """ 
        Attributes:
            kind (AutomationTriggerKind):
            after_hours (int | Unset):
            also (list[AutomationTrigger] | Unset):
            answer_op (AutomationTriggerAnswerOp | Unset):
            answers (list[int] | Unset):
            at (str | Unset):
            collections (list[str] | Unset):
            confirmed_only (bool | Unset):
            contact_time (bool | Unset):
            direction (AutomationTriggerDirection | Unset):
            end_reasons (list[AutomationTriggerEndReasonsItem] | Unset):
            event (str | Unset):
            exit_on_leave (bool | Unset):
            exit_on_reply (bool | Unset):
            field (str | Unset):
            first_only (bool | Unset):
            form_id (str | Unset):
            include_imports (bool | Unset):
            lead_form_id (str | Unset):
            list_id (str | Unset):
            milestone_id (str | Unset):
            min_total (int | Unset):
            offset_days (int | Unset):
            on_change (bool | Unset):
            once_in_days (int | Unset):
            poll (str | Unset):
            product_id (str | Unset):
            products (list[str] | Unset):
            program_id (str | Unset):
            reentry (str | Unset):
            reentry_days (int | Unset):
            segment_id (str | Unset):
            skip_days (int | Unset):
            source (str | Unset):
            store (str | Unset):
            tag (str | Unset):
            threshold (int | Unset):
            tier_id (str | Unset):
            url_contains (str | Unset):
            yearly (bool | Unset):
     """

    kind: AutomationTriggerKind
    after_hours: int | Unset = UNSET
    also: list[AutomationTrigger] | Unset = UNSET
    answer_op: AutomationTriggerAnswerOp | Unset = UNSET
    answers: list[int] | Unset = UNSET
    at: str | Unset = UNSET
    collections: list[str] | Unset = UNSET
    confirmed_only: bool | Unset = UNSET
    contact_time: bool | Unset = UNSET
    direction: AutomationTriggerDirection | Unset = UNSET
    end_reasons: list[AutomationTriggerEndReasonsItem] | Unset = UNSET
    event: str | Unset = UNSET
    exit_on_leave: bool | Unset = UNSET
    exit_on_reply: bool | Unset = UNSET
    field: str | Unset = UNSET
    first_only: bool | Unset = UNSET
    form_id: str | Unset = UNSET
    include_imports: bool | Unset = UNSET
    lead_form_id: str | Unset = UNSET
    list_id: str | Unset = UNSET
    milestone_id: str | Unset = UNSET
    min_total: int | Unset = UNSET
    offset_days: int | Unset = UNSET
    on_change: bool | Unset = UNSET
    once_in_days: int | Unset = UNSET
    poll: str | Unset = UNSET
    product_id: str | Unset = UNSET
    products: list[str] | Unset = UNSET
    program_id: str | Unset = UNSET
    reentry: str | Unset = UNSET
    reentry_days: int | Unset = UNSET
    segment_id: str | Unset = UNSET
    skip_days: int | Unset = UNSET
    source: str | Unset = UNSET
    store: str | Unset = UNSET
    tag: str | Unset = UNSET
    threshold: int | Unset = UNSET
    tier_id: str | Unset = UNSET
    url_contains: str | Unset = UNSET
    yearly: bool | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        kind = self.kind.value

        after_hours = self.after_hours

        also: list[dict[str, Any]] | Unset = UNSET
        if not isinstance(self.also, Unset):
            also = []
            for also_item_data in self.also:
                also_item = also_item_data.to_dict()
                also.append(also_item)



        answer_op: str | Unset = UNSET
        if not isinstance(self.answer_op, Unset):
            answer_op = self.answer_op.value


        answers: list[int] | Unset = UNSET
        if not isinstance(self.answers, Unset):
            answers = self.answers



        at = self.at

        collections: list[str] | Unset = UNSET
        if not isinstance(self.collections, Unset):
            collections = self.collections



        confirmed_only = self.confirmed_only

        contact_time = self.contact_time

        direction: str | Unset = UNSET
        if not isinstance(self.direction, Unset):
            direction = self.direction.value


        end_reasons: list[str] | Unset = UNSET
        if not isinstance(self.end_reasons, Unset):
            end_reasons = []
            for end_reasons_item_data in self.end_reasons:
                end_reasons_item = end_reasons_item_data.value
                end_reasons.append(end_reasons_item)



        event = self.event

        exit_on_leave = self.exit_on_leave

        exit_on_reply = self.exit_on_reply

        field = self.field

        first_only = self.first_only

        form_id = self.form_id

        include_imports = self.include_imports

        lead_form_id = self.lead_form_id

        list_id = self.list_id

        milestone_id = self.milestone_id

        min_total = self.min_total

        offset_days = self.offset_days

        on_change = self.on_change

        once_in_days = self.once_in_days

        poll = self.poll

        product_id = self.product_id

        products: list[str] | Unset = UNSET
        if not isinstance(self.products, Unset):
            products = self.products



        program_id = self.program_id

        reentry = self.reentry

        reentry_days = self.reentry_days

        segment_id = self.segment_id

        skip_days = self.skip_days

        source = self.source

        store = self.store

        tag = self.tag

        threshold = self.threshold

        tier_id = self.tier_id

        url_contains = self.url_contains

        yearly = self.yearly


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "kind": kind,
        })
        if after_hours is not UNSET:
            field_dict["afterHours"] = after_hours
        if also is not UNSET:
            field_dict["also"] = also
        if answer_op is not UNSET:
            field_dict["answerOp"] = answer_op
        if answers is not UNSET:
            field_dict["answers"] = answers
        if at is not UNSET:
            field_dict["at"] = at
        if collections is not UNSET:
            field_dict["collections"] = collections
        if confirmed_only is not UNSET:
            field_dict["confirmedOnly"] = confirmed_only
        if contact_time is not UNSET:
            field_dict["contactTime"] = contact_time
        if direction is not UNSET:
            field_dict["direction"] = direction
        if end_reasons is not UNSET:
            field_dict["endReasons"] = end_reasons
        if event is not UNSET:
            field_dict["event"] = event
        if exit_on_leave is not UNSET:
            field_dict["exitOnLeave"] = exit_on_leave
        if exit_on_reply is not UNSET:
            field_dict["exitOnReply"] = exit_on_reply
        if field is not UNSET:
            field_dict["field"] = field
        if first_only is not UNSET:
            field_dict["firstOnly"] = first_only
        if form_id is not UNSET:
            field_dict["formId"] = form_id
        if include_imports is not UNSET:
            field_dict["includeImports"] = include_imports
        if lead_form_id is not UNSET:
            field_dict["leadFormId"] = lead_form_id
        if list_id is not UNSET:
            field_dict["listId"] = list_id
        if milestone_id is not UNSET:
            field_dict["milestoneId"] = milestone_id
        if min_total is not UNSET:
            field_dict["minTotal"] = min_total
        if offset_days is not UNSET:
            field_dict["offsetDays"] = offset_days
        if on_change is not UNSET:
            field_dict["onChange"] = on_change
        if once_in_days is not UNSET:
            field_dict["onceInDays"] = once_in_days
        if poll is not UNSET:
            field_dict["poll"] = poll
        if product_id is not UNSET:
            field_dict["productId"] = product_id
        if products is not UNSET:
            field_dict["products"] = products
        if program_id is not UNSET:
            field_dict["programId"] = program_id
        if reentry is not UNSET:
            field_dict["reentry"] = reentry
        if reentry_days is not UNSET:
            field_dict["reentryDays"] = reentry_days
        if segment_id is not UNSET:
            field_dict["segmentId"] = segment_id
        if skip_days is not UNSET:
            field_dict["skipDays"] = skip_days
        if source is not UNSET:
            field_dict["source"] = source
        if store is not UNSET:
            field_dict["store"] = store
        if tag is not UNSET:
            field_dict["tag"] = tag
        if threshold is not UNSET:
            field_dict["threshold"] = threshold
        if tier_id is not UNSET:
            field_dict["tierId"] = tier_id
        if url_contains is not UNSET:
            field_dict["urlContains"] = url_contains
        if yearly is not UNSET:
            field_dict["yearly"] = yearly

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        d = dict(src_dict)
        kind = AutomationTriggerKind(d.pop("kind"))




        after_hours = d.pop("afterHours", UNSET)

        _also = d.pop("also", UNSET)
        also: list[AutomationTrigger] | Unset = UNSET
        if _also is not UNSET:
            also = []
            for also_item_data in _also:
                also_item = AutomationTrigger.from_dict(also_item_data)



                also.append(also_item)


        _answer_op = d.pop("answerOp", UNSET)
        answer_op: AutomationTriggerAnswerOp | Unset
        if isinstance(_answer_op,  Unset):
            answer_op = UNSET
        else:
            answer_op = AutomationTriggerAnswerOp(_answer_op)




        answers = cast(list[int], d.pop("answers", UNSET))


        at = d.pop("at", UNSET)

        collections = cast(list[str], d.pop("collections", UNSET))


        confirmed_only = d.pop("confirmedOnly", UNSET)

        contact_time = d.pop("contactTime", UNSET)

        _direction = d.pop("direction", UNSET)
        direction: AutomationTriggerDirection | Unset
        if isinstance(_direction,  Unset):
            direction = UNSET
        else:
            direction = AutomationTriggerDirection(_direction)




        _end_reasons = d.pop("endReasons", UNSET)
        end_reasons: list[AutomationTriggerEndReasonsItem] | Unset = UNSET
        if _end_reasons is not UNSET:
            end_reasons = []
            for end_reasons_item_data in _end_reasons:
                end_reasons_item = AutomationTriggerEndReasonsItem(end_reasons_item_data)



                end_reasons.append(end_reasons_item)


        event = d.pop("event", UNSET)

        exit_on_leave = d.pop("exitOnLeave", UNSET)

        exit_on_reply = d.pop("exitOnReply", UNSET)

        field = d.pop("field", UNSET)

        first_only = d.pop("firstOnly", UNSET)

        form_id = d.pop("formId", UNSET)

        include_imports = d.pop("includeImports", UNSET)

        lead_form_id = d.pop("leadFormId", UNSET)

        list_id = d.pop("listId", UNSET)

        milestone_id = d.pop("milestoneId", UNSET)

        min_total = d.pop("minTotal", UNSET)

        offset_days = d.pop("offsetDays", UNSET)

        on_change = d.pop("onChange", UNSET)

        once_in_days = d.pop("onceInDays", UNSET)

        poll = d.pop("poll", UNSET)

        product_id = d.pop("productId", UNSET)

        products = cast(list[str], d.pop("products", UNSET))


        program_id = d.pop("programId", UNSET)

        reentry = d.pop("reentry", UNSET)

        reentry_days = d.pop("reentryDays", UNSET)

        segment_id = d.pop("segmentId", UNSET)

        skip_days = d.pop("skipDays", UNSET)

        source = d.pop("source", UNSET)

        store = d.pop("store", UNSET)

        tag = d.pop("tag", UNSET)

        threshold = d.pop("threshold", UNSET)

        tier_id = d.pop("tierId", UNSET)

        url_contains = d.pop("urlContains", UNSET)

        yearly = d.pop("yearly", UNSET)

        automation_trigger = cls(
            kind=kind,
            after_hours=after_hours,
            also=also,
            answer_op=answer_op,
            answers=answers,
            at=at,
            collections=collections,
            confirmed_only=confirmed_only,
            contact_time=contact_time,
            direction=direction,
            end_reasons=end_reasons,
            event=event,
            exit_on_leave=exit_on_leave,
            exit_on_reply=exit_on_reply,
            field=field,
            first_only=first_only,
            form_id=form_id,
            include_imports=include_imports,
            lead_form_id=lead_form_id,
            list_id=list_id,
            milestone_id=milestone_id,
            min_total=min_total,
            offset_days=offset_days,
            on_change=on_change,
            once_in_days=once_in_days,
            poll=poll,
            product_id=product_id,
            products=products,
            program_id=program_id,
            reentry=reentry,
            reentry_days=reentry_days,
            segment_id=segment_id,
            skip_days=skip_days,
            source=source,
            store=store,
            tag=tag,
            threshold=threshold,
            tier_id=tier_id,
            url_contains=url_contains,
            yearly=yearly,
        )

        return automation_trigger

