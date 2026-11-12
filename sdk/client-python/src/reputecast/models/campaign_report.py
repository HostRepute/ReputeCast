from __future__ import annotations

from collections.abc import Mapping
from typing import Any, TypeVar, BinaryIO, TextIO, TYPE_CHECKING, Generator

from attrs import define as _attrs_define
from attrs import field as _attrs_field

from ..types import UNSET, Unset

from ..types import UNSET, Unset
from typing import cast

if TYPE_CHECKING:
  from ..models.report_ab_test import ReportABTest
  from ..models.report_bounce_table import ReportBounceTable
  from ..models.report_breakdown import ReportBreakdown
  from ..models.report_campaign_info import ReportCampaignInfo
  from ..models.report_click_map import ReportClickMap
  from ..models.report_comparisons import ReportComparisons
  from ..models.report_engagement import ReportEngagement
  from ..models.report_growth import ReportGrowth
  from ..models.report_headline import ReportHeadline
  from ..models.report_ingest_backlog import ReportIngestBacklog
  from ..models.report_insight import ReportInsight
  from ..models.report_languages import ReportLanguages
  from ..models.report_mailbox_row import ReportMailboxRow
  from ..models.report_not_sent import ReportNotSent
  from ..models.report_provider_table import ReportProviderTable
  from ..models.report_unsubscribes import ReportUnsubscribes





T = TypeVar("T", bound="CampaignReport")



@_attrs_define
class CampaignReport:
    """ 
        Attributes:
            bounces (ReportBounceTable):
            breakdown (ReportBreakdown):
            campaign (ReportCampaignInfo):
            click_map (ReportClickMap):
            comparisons (ReportComparisons):
            engagement (ReportEngagement):
            growth (ReportGrowth):
            headline (ReportHeadline):
            insights (list[ReportInsight]):
            mailboxes (list[ReportMailboxRow]):
            providers (ReportProviderTable):
            unsubscribes (ReportUnsubscribes):
            ab_test (ReportABTest | Unset):
            backlog (ReportIngestBacklog | Unset):
            languages (ReportLanguages | Unset):
            not_sent (ReportNotSent | Unset):
     """

    bounces: ReportBounceTable
    breakdown: ReportBreakdown
    campaign: ReportCampaignInfo
    click_map: ReportClickMap
    comparisons: ReportComparisons
    engagement: ReportEngagement
    growth: ReportGrowth
    headline: ReportHeadline
    insights: list[ReportInsight]
    mailboxes: list[ReportMailboxRow]
    providers: ReportProviderTable
    unsubscribes: ReportUnsubscribes
    ab_test: ReportABTest | Unset = UNSET
    backlog: ReportIngestBacklog | Unset = UNSET
    languages: ReportLanguages | Unset = UNSET
    not_sent: ReportNotSent | Unset = UNSET





    def to_dict(self) -> dict[str, Any]:
        from ..models.report_ab_test import ReportABTest # noqa: PLC0415
        from ..models.report_bounce_table import ReportBounceTable # noqa: PLC0415
        from ..models.report_breakdown import ReportBreakdown # noqa: PLC0415
        from ..models.report_campaign_info import ReportCampaignInfo # noqa: PLC0415
        from ..models.report_click_map import ReportClickMap # noqa: PLC0415
        from ..models.report_comparisons import ReportComparisons # noqa: PLC0415
        from ..models.report_engagement import ReportEngagement # noqa: PLC0415
        from ..models.report_growth import ReportGrowth # noqa: PLC0415
        from ..models.report_headline import ReportHeadline # noqa: PLC0415
        from ..models.report_ingest_backlog import ReportIngestBacklog # noqa: PLC0415
        from ..models.report_insight import ReportInsight # noqa: PLC0415
        from ..models.report_languages import ReportLanguages # noqa: PLC0415
        from ..models.report_mailbox_row import ReportMailboxRow # noqa: PLC0415
        from ..models.report_not_sent import ReportNotSent # noqa: PLC0415
        from ..models.report_provider_table import ReportProviderTable # noqa: PLC0415
        from ..models.report_unsubscribes import ReportUnsubscribes # noqa: PLC0415
        bounces = self.bounces.to_dict()

        breakdown = self.breakdown.to_dict()

        campaign = self.campaign.to_dict()

        click_map = self.click_map.to_dict()

        comparisons = self.comparisons.to_dict()

        engagement = self.engagement.to_dict()

        growth = self.growth.to_dict()

        headline = self.headline.to_dict()

        insights = []
        for insights_item_data in self.insights:
            insights_item = insights_item_data.to_dict()
            insights.append(insights_item)



        mailboxes = []
        for mailboxes_item_data in self.mailboxes:
            mailboxes_item = mailboxes_item_data.to_dict()
            mailboxes.append(mailboxes_item)



        providers = self.providers.to_dict()

        unsubscribes = self.unsubscribes.to_dict()

        ab_test: dict[str, Any] | Unset = UNSET
        if not isinstance(self.ab_test, Unset):
            ab_test = self.ab_test.to_dict()

        backlog: dict[str, Any] | Unset = UNSET
        if not isinstance(self.backlog, Unset):
            backlog = self.backlog.to_dict()

        languages: dict[str, Any] | Unset = UNSET
        if not isinstance(self.languages, Unset):
            languages = self.languages.to_dict()

        not_sent: dict[str, Any] | Unset = UNSET
        if not isinstance(self.not_sent, Unset):
            not_sent = self.not_sent.to_dict()


        field_dict: dict[str, Any] = {}

        field_dict.update({
            "bounces": bounces,
            "breakdown": breakdown,
            "campaign": campaign,
            "clickMap": click_map,
            "comparisons": comparisons,
            "engagement": engagement,
            "growth": growth,
            "headline": headline,
            "insights": insights,
            "mailboxes": mailboxes,
            "providers": providers,
            "unsubscribes": unsubscribes,
        })
        if ab_test is not UNSET:
            field_dict["abTest"] = ab_test
        if backlog is not UNSET:
            field_dict["backlog"] = backlog
        if languages is not UNSET:
            field_dict["languages"] = languages
        if not_sent is not UNSET:
            field_dict["notSent"] = not_sent

        return field_dict



    @classmethod
    def from_dict(cls: type[T], src_dict: Mapping[str, Any]) -> T:
        from ..models.report_ab_test import ReportABTest # noqa: PLC0415
        from ..models.report_bounce_table import ReportBounceTable # noqa: PLC0415
        from ..models.report_breakdown import ReportBreakdown # noqa: PLC0415
        from ..models.report_campaign_info import ReportCampaignInfo # noqa: PLC0415
        from ..models.report_click_map import ReportClickMap # noqa: PLC0415
        from ..models.report_comparisons import ReportComparisons # noqa: PLC0415
        from ..models.report_engagement import ReportEngagement # noqa: PLC0415
        from ..models.report_growth import ReportGrowth # noqa: PLC0415
        from ..models.report_headline import ReportHeadline # noqa: PLC0415
        from ..models.report_ingest_backlog import ReportIngestBacklog # noqa: PLC0415
        from ..models.report_insight import ReportInsight # noqa: PLC0415
        from ..models.report_languages import ReportLanguages # noqa: PLC0415
        from ..models.report_mailbox_row import ReportMailboxRow # noqa: PLC0415
        from ..models.report_not_sent import ReportNotSent # noqa: PLC0415
        from ..models.report_provider_table import ReportProviderTable # noqa: PLC0415
        from ..models.report_unsubscribes import ReportUnsubscribes # noqa: PLC0415
        d = dict(src_dict)
        bounces = ReportBounceTable.from_dict(d.pop("bounces"))




        breakdown = ReportBreakdown.from_dict(d.pop("breakdown"))




        campaign = ReportCampaignInfo.from_dict(d.pop("campaign"))




        click_map = ReportClickMap.from_dict(d.pop("clickMap"))




        comparisons = ReportComparisons.from_dict(d.pop("comparisons"))




        engagement = ReportEngagement.from_dict(d.pop("engagement"))




        growth = ReportGrowth.from_dict(d.pop("growth"))




        headline = ReportHeadline.from_dict(d.pop("headline"))




        insights = []
        _insights = d.pop("insights")
        for insights_item_data in (_insights):
            insights_item = ReportInsight.from_dict(insights_item_data)



            insights.append(insights_item)


        mailboxes = []
        _mailboxes = d.pop("mailboxes")
        for mailboxes_item_data in (_mailboxes):
            mailboxes_item = ReportMailboxRow.from_dict(mailboxes_item_data)



            mailboxes.append(mailboxes_item)


        providers = ReportProviderTable.from_dict(d.pop("providers"))




        unsubscribes = ReportUnsubscribes.from_dict(d.pop("unsubscribes"))




        _ab_test = d.pop("abTest", UNSET)
        ab_test: ReportABTest | Unset
        if isinstance(_ab_test,  Unset):
            ab_test = UNSET
        else:
            ab_test = ReportABTest.from_dict(_ab_test)




        _backlog = d.pop("backlog", UNSET)
        backlog: ReportIngestBacklog | Unset
        if isinstance(_backlog,  Unset):
            backlog = UNSET
        else:
            backlog = ReportIngestBacklog.from_dict(_backlog)




        _languages = d.pop("languages", UNSET)
        languages: ReportLanguages | Unset
        if isinstance(_languages,  Unset):
            languages = UNSET
        else:
            languages = ReportLanguages.from_dict(_languages)




        _not_sent = d.pop("notSent", UNSET)
        not_sent: ReportNotSent | Unset
        if isinstance(_not_sent,  Unset):
            not_sent = UNSET
        else:
            not_sent = ReportNotSent.from_dict(_not_sent)




        campaign_report = cls(
            bounces=bounces,
            breakdown=breakdown,
            campaign=campaign,
            click_map=click_map,
            comparisons=comparisons,
            engagement=engagement,
            growth=growth,
            headline=headline,
            insights=insights,
            mailboxes=mailboxes,
            providers=providers,
            unsubscribes=unsubscribes,
            ab_test=ab_test,
            backlog=backlog,
            languages=languages,
            not_sent=not_sent,
        )

        return campaign_report

