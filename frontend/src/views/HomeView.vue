<template>
  <div class="fuchibbol-layout">
    <!-- Top Bar -->
    <div class="topbar">
      <div class="logo">
        <div class="logo-dot"></div>
        <span class="logo-text">FUCHIBBOL</span>
      </div>
      <div class="topbar-right">
        <div class="user-badge" v-if="currentUser">
          <div class="user-avatar">{{ currentUser.username.substring(0, 1).toUpperCase() }}</div>
          <span class="user-name">@{{ currentUser.username }}</span>
        </div>
        <button @click="installApp" class="follow-btn" style="padding: 5px 10px; margin-right: 8px; font-size: 11px;">Instalar App</button>
        <router-link v-if="currentUser && ['admin', 'streamer'].includes(currentUser.role)" to="/admin" class="exit-btn" style="text-decoration:none; margin-right: 8px;">Panel</router-link>
        <button v-if="currentUser" @click="logout" class="exit-btn">Salir</button>
        <router-link v-else to="/login" class="exit-btn" style="text-decoration:none; color: #00e87a; border-color: #00e87a;">Entrar</router-link>
      </div>
    </div>

    <template v-if="channelInfo">
      <div class="home-grid">
        <div class="home-main">
          <!-- Video Player -->
      <div class="video-wrap">
        <VideoPlayer :key="playerKey" :streamUrl="playbackUrl" :isLive="channelInfo.is_live" :channelId="channelInfo.id" />
        <div class="video-overlay" v-if="channelInfo.is_live"></div>
        <div class="video-top" v-if="channelInfo.is_live">
          <div class="viewers">
            <div class="viewers-dot"></div>
            <span class="viewers-count">{{ viewerCount }} espectadores</span>
          </div>
        </div>
      </div>

      <!-- Channel Row -->
      <div class="channel-row">
        <img v-if="channelInfo.logo_url" class="channel-icon" :src="channelInfo.logo_url" alt="Logo" />
        <div v-else class="channel-icon">{{ channelOwner.substring(0, 2).toUpperCase() }}</div>
        <div class="channel-info">
          <div class="channel-name">{{ channelInfo.name }}</div>
          <div class="channel-meta">
            <span class="channel-handle">@{{ channelOwner }}</span>
            <span class="channel-followers">· {{ followerCount }} seguidores</span>
            <span class="channel-live-tag" v-if="channelInfo.is_live">VIVO</span>
          </div>
        </div>
        <button class="follow-btn" @click="toggleFollow" :style="isFollowing ? 'background: rgba(0,232,122,0.1)' : ''">
          {{ isFollowing ? 'Siguiendo' : 'Seguir' }}
        </button>
      </div>

      <!-- Cast to TV Button -->
      <div v-if="channelInfo.is_live" style="display: flex; align-items: center; gap: 10px; padding: 8px 0 4px 0;">
        <button @click="triggerCastTV" :class="isCasting ? 'cast-tv-btn active' : 'cast-tv-btn'">
          <svg viewBox="0 0 24 24" width="15" height="15" fill="currentColor">
            <path d="M1 18v3h3c0-1.66-1.34-3-3-3zm0-4v2c2.76 0 5 2.24 5 5h2c0-3.87-3.13-7-7-7zm0-4v2c4.97 0 9 4.03 9 9h2c0-6.08-4.93-11-11-11zm20-7H3c-1.1 0-2 .9-2 2v3h2V5h18v14h-7v2h7c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2z"/>
          </svg>
          {{ isCasting ? 'Transmitiendo en TV...' : 'Ver en Smart TV' }}
        </button>
      </div>

      <!-- Mobile Donations and Ranking Block -->
      <div class="hide-desktop">
        <!-- Donations Panel -->
        <div v-if="channelInfo && (channelInfo.yape_number || channelInfo.paypal_link)" class="donations-panel" style="margin-top: 16px; padding: 14px; background: #13131f; border-radius: 8px; border: 0.5px solid #1a1a28;">
          <div style="font-size: 11px; font-weight: 700; color: #606080; letter-spacing: 0.5px; margin-bottom: 10px; text-transform: uppercase;">Apoya el Canal (Donaciones)</div>
          <div v-if="channelInfo.donation_message" style="font-size: 13px; color: #c0c0d8; margin-bottom: 8px; line-height: 1.5; white-space: pre-line;">
            <span>{{ channelInfo.donation_message }}</span>
            <button v-if="channelInfo.donation_long_message" @click="isDonationExpanded = !isDonationExpanded" style="background: none; border: none; color: #00e87a; font-weight: 600; cursor: pointer; margin-left: 6px; padding: 0; font-size: 13px; text-decoration: underline;">
              {{ isDonationExpanded ? 'Ver menos' : 'Ver más' }}
            </button>
          </div>
          <div v-if="isDonationExpanded && channelInfo.donation_long_message" style="font-size: 12.5px; color: #9595b0; margin-bottom: 12px; line-height: 1.5; white-space: pre-line; padding-left: 6px; border-left: 2px solid #2a2a40; transition: all 0.3s;">
            {{ channelInfo.donation_long_message }}
          </div>
          <div style="display: flex; flex-wrap: wrap; gap: 12px; align-items: center; margin-top: 10px;">
            <!-- Yape -->
            <div v-if="channelInfo.yape_number" style="display: flex; align-items: center; gap: 8px; background: #1a1a28; padding: 6px 12px; border-radius: 6px; border: 0.5px solid #2a2a40;">
              <span style="font-size: 12px; font-weight: 700; color: #00e87a;">Yape:</span>
              <span style="font-size: 12px; color: #c0c0d8;">{{ channelInfo.yape_number }}</span>
              <button @click="copyYapeNumber" style="background: none; border: none; color: #00e87a; cursor: pointer; display: flex; align-items: center; padding: 2px;" title="Copiar número de Yape">
                <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-copy"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
              </button>
            </div>
            <!-- PayPal -->
            <a v-if="channelInfo.paypal_link" :href="channelInfo.paypal_link" target="_blank" style="display: inline-flex; align-items: center; gap: 8px; background: #0070ba; color: #fff; padding: 6px 14px; border-radius: 6px; text-decoration: none; font-size: 12px; font-weight: 700; transition: background 0.2s;" onmouseover="this.style.background='#005ea6'" onmouseout="this.style.background='#0070ba'">
              <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" fill="currentColor" viewBox="0 0 24 24" style="margin-right: 2px;"><path d="M7.076 21.337H2.47L5.58 1.578h9.72c2.475 0 4.417.587 5.561 1.674 1.096 1.042 1.48 2.535 1.15 4.63-.647 4.119-3.235 6.45-6.892 6.45H10.45l-1.393 8.75c-.073.454-.424.81-1.981.81zM14.61 5.518H9.378l-1.333 8.388h3.948c2.203 0 3.738-1.282 4.103-3.606.333-2.115-.595-3.328-2.613-3.328z" /></svg>
              Donar con PayPal
            </a>
            <!-- Report button -->
            <button @click="showReportModal = true" style="margin-left: auto; background: rgba(0,232,122,0.1); color: #00e87a; border: 0.5px solid rgba(0,232,122,0.3); padding: 6px 14px; border-radius: 6px; font-size: 12px; font-weight: 700; cursor: pointer; transition: all 0.2s;" onmouseover="this.style.background='rgba(0,232,122,0.2)'" onmouseout="this.style.background='rgba(0,232,122,0.1)'">
              Reportar Donación
            </button>
          </div>
        </div>

        <!-- Top Donors Ranking -->
        <div v-if="donationRanking && donationRanking.length > 0" class="donations-panel" style="margin-top: 12px; padding: 14px; background: #13131f; border-radius: 8px; border: 0.5px solid #1a1a28;">
          <div style="font-size: 11px; font-weight: 700; color: #606080; letter-spacing: 0.5px; margin-bottom: 10px; text-transform: uppercase;">Top Donadores 🏆</div>
          <div style="display: flex; flex-direction: column; gap: 8px;">
            <div v-for="(donor, idx) in donationRanking" :key="idx" style="display: flex; justify-content: space-between; align-items: center; background: #1a1a28; padding: 6px 12px; border-radius: 6px; border: 0.5px solid #2a2a40;">
              <div style="display: flex; align-items: center; gap: 8px;">
                <span style="font-weight: 700; color: #606080; font-size: 12px;">#{{ idx + 1 }}</span>
                <span style="font-size: 13px; color: #c0c0d8; font-weight: 600;">{{ donor.donor_name }}</span>
              </div>
              <span style="font-size: 13px; font-weight: 700; color: #00e87a;">S/. {{ donor.total_amount.toFixed(2) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- About -->
      <div class="section-label">SOBRE EL CANAL</div>
      <div class="streamer-bio">{{ channelInfo.description || 'Este canal no tiene descripción.' }}</div>

        </div>
        <div class="home-sidebar">
      <!-- Match Banner (Agenda) -->
      <div class="match-banner" v-for="(event, index) in parsedAgendaEvents" :key="index">
        <div class="match-flags">{{ event.flags }}</div>
        <div class="match-info">
          <div class="match-title">{{ event.teams }}</div>
          <div class="match-time" :style="event.isActive ? 'color: #e83d00;' : ''">{{ event.time }}</div>
        </div>
        <div style="display: flex; flex-direction: column; align-items: flex-end; gap: 6px;">
          <div class="match-pill" v-if="event.isActive" style="color: #e83d00; background: rgba(232,61,0,0.1); border-color: rgba(232,61,0,0.3);">AHORA</div>
          <div class="match-pill" v-else>PRÓXIMO</div>
          <button @click="shareMatch(event)" class="share-match-btn" title="Compartir partido">
            <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="5" r="3"></circle><circle cx="6" cy="12" r="3"></circle><circle cx="18" cy="19" r="3"></circle><line x1="8.59" y1="13.51" x2="15.42" y2="17.49"></line><line x1="15.41" y1="6.51" x2="8.59" y2="10.49"></line></svg>
            Compartir
          </button>
        </div>
      </div>


          <!-- Desktop Donations and Ranking Block -->
          <div class="hide-mobile" style="margin-bottom: 12px; padding: 0 16px;">
            <!-- Donations Panel -->
            <div v-if="channelInfo && (channelInfo.yape_number || channelInfo.paypal_link)" class="donations-panel" style="padding: 14px; background: #13131f; border-radius: 8px; border: 0.5px solid #1a1a28;">
              <div style="font-size: 11px; font-weight: 700; color: #606080; letter-spacing: 0.5px; margin-bottom: 10px; text-transform: uppercase;">Apoya el Canal (Donaciones)</div>
              <div v-if="channelInfo.donation_message" style="font-size: 13px; color: #c0c0d8; margin-bottom: 8px; line-height: 1.5; white-space: pre-line;">
                <span>{{ channelInfo.donation_message }}</span>
                <button v-if="channelInfo.donation_long_message" @click="isDonationExpanded = !isDonationExpanded" style="background: none; border: none; color: #00e87a; font-weight: 600; cursor: pointer; margin-left: 6px; padding: 0; font-size: 13px; text-decoration: underline;">
                  {{ isDonationExpanded ? 'Ver menos' : 'Ver más' }}
                </button>
              </div>
              <div v-if="isDonationExpanded && channelInfo.donation_long_message" style="font-size: 12.5px; color: #9595b0; margin-bottom: 12px; line-height: 1.5; white-space: pre-line; padding-left: 6px; border-left: 2px solid #2a2a40; transition: all 0.3s;">
                {{ channelInfo.donation_long_message }}
              </div>
              <div style="display: flex; flex-wrap: wrap; gap: 8px; align-items: center; margin-top: 10px;">
                <!-- Yape -->
                <div v-if="channelInfo.yape_number" style="display: flex; align-items: center; gap: 6px; background: #1a1a28; padding: 4px 8px; border-radius: 6px; border: 0.5px solid #2a2a40;">
                  <span style="font-size: 11px; font-weight: 700; color: #00e87a;">Yape:</span>
                  <span style="font-size: 11px; color: #c0c0d8;">{{ channelInfo.yape_number }}</span>
                  <button @click="copyYapeNumber" style="background: none; border: none; color: #00e87a; cursor: pointer; display: flex; align-items: center; padding: 2px;" title="Copiar número de Yape">
                    <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="feather feather-copy"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"></rect><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"></path></svg>
                  </button>
                </div>
                <!-- PayPal -->
                <a v-if="channelInfo.paypal_link" :href="channelInfo.paypal_link" target="_blank" style="display: inline-flex; align-items: center; gap: 6px; background: #0070ba; color: #fff; padding: 4px 10px; border-radius: 6px; text-decoration: none; font-size: 11px; font-weight: 700; transition: background 0.2s;" onmouseover="this.style.background='#005ea6'" onmouseout="this.style.background='#0070ba'">
                  <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" fill="currentColor" viewBox="0 0 24 24" style="margin-right: 2px;"><path d="M7.076 21.337H2.47L5.58 1.578h9.72c2.475 0 4.417.587 5.561 1.674 1.096 1.042 1.48 2.535 1.15 4.63-.647 4.119-3.235 6.45-6.892 6.45H10.45l-1.393 8.75c-.073.454-.424.81-1.981.81zM14.61 5.518H9.378l-1.333 8.388h3.948c2.203 0 3.738-1.282 4.103-3.606.333-2.115-.595-3.328-2.613-3.328z" /></svg>
                  PayPal
                </a>
                <!-- Report button -->
                <button @click="showReportModal = true" style="margin-left: auto; background: rgba(0,232,122,0.1); color: #00e87a; border: 0.5px solid rgba(0,232,122,0.3); padding: 4px 10px; border-radius: 6px; font-size: 11px; font-weight: 700; cursor: pointer; transition: all 0.2s;" onmouseover="this.style.background='rgba(0,232,122,0.2)'" onmouseout="this.style.background='rgba(0,232,122,0.1)'">
                  Reportar
                </button>
              </div>
            </div>

            <!-- Top Donors Ranking -->
            <div v-if="donationRanking && donationRanking.length > 0" class="donations-panel" style="margin-top: 10px; padding: 14px; background: #13131f; border-radius: 8px; border: 0.5px solid #1a1a28;">
              <div style="font-size: 11px; font-weight: 700; color: #606080; letter-spacing: 0.5px; margin-bottom: 10px; text-transform: uppercase;">Top Donadores 🏆</div>
              <div style="display: flex; flex-direction: column; gap: 6px;">
                <div v-for="(donor, idx) in donationRanking" :key="idx" style="display: flex; justify-content: space-between; align-items: center; background: #1a1a28; padding: 5px 10px; border-radius: 6px; border: 0.5px solid #2a2a40;">
                  <div style="display: flex; align-items: center; gap: 6px;">
                    <span style="font-weight: 700; color: #606080; font-size: 11px;">#{{ idx + 1 }}</span>
                    <span style="font-size: 12px; color: #c0c0d8; font-weight: 600;">{{ donor.donor_name }}</span>
                  </div>
                  <span style="font-size: 12px; font-weight: 700; color: #00e87a;">S/. {{ donor.total_amount.toFixed(2) }}</span>
                </div>
              </div>
            </div>
          </div>

          <!-- Chat -->
          <ChatBox 
            :channelId="channelInfo.id" 
            :currentUser="currentUser" 
            @viewer-update="handleViewerUpdate" 
            @stream-reload="handleStreamReload"
          />
        </div>
      </div>
    </template>

    <div v-else class="video-placeholder" style="flex-direction: column; height: 100vh;">
      <div class="logo-dot"></div>
      <div style="margin-top:15px; color:#c0c0d8; font-size:14px;">Cargando estadio...</div>
    </div>
    
    <!-- WHATSAPP FLOATING BUTTON -->
    <a 
      v-if="channelInfo && channelInfo.whatsapp_link" 
      :href="channelInfo.whatsapp_link" 
      target="_blank" 
      class="wsp-float-btn"
      title="Únete a nuestro grupo de WhatsApp"
    >
      <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
        <path d="M17.472 14.382c-.297-.149-1.758-.867-2.03-.967-.273-.099-.471-.148-.67.15-.197.297-.767.966-.94 1.164-.173.199-.347.223-.644.075-.297-.15-1.255-.463-2.39-1.475-.883-.788-1.48-1.761-1.653-2.059-.173-.297-.018-.458.13-.606.134-.133.298-.347.446-.52.149-.174.198-.298.298-.497.099-.198.05-.371-.025-.52-.075-.149-.669-1.612-.916-2.207-.242-.579-.487-.5-.669-.51-.173-.008-.371-.01-.57-.01-.198 0-.52.074-.792.372-.272.297-1.04 1.016-1.04 2.479 0 1.462 1.065 2.875 1.213 3.074.149.198 2.096 3.2 5.077 4.487.709.306 1.262.489 1.694.625.712.227 1.36.195 1.871.118.571-.085 1.758-.719 2.006-1.413.248-.694.248-1.289.173-1.413-.074-.124-.272-.198-.57-.347m-5.421 7.403h-.004a9.87 9.87 0 01-5.031-1.378l-.361-.214-3.741.982.998-3.648-.235-.374a9.86 9.86 0 01-1.51-5.26c.001-5.45 4.436-9.884 9.888-9.884 2.64 0 5.122 1.03 6.988 2.898a9.825 9.825 0 012.893 6.994c-.003 5.45-4.437 9.884-9.885 9.884m8.413-18.297A11.815 11.815 0 0012.05 0C5.495 0 .16 5.335.157 11.892c0 2.096.547 4.142 1.588 5.945L.057 24l6.305-1.654a11.882 11.882 0 005.683 1.448h.005c6.554 0 11.89-5.335 11.893-11.893a11.821 11.821 0 00-3.48-8.413z"/>
      </svg>
    </a>

    <!-- TIKTOK FLOATING BUTTON -->
    <a 
      v-if="channelInfo && channelInfo.tiktok_link" 
      :href="channelInfo.tiktok_link" 
      target="_blank" 
      class="tiktok-float-btn"
      title="Síguenos en TikTok"
    >
      <svg viewBox="0 0 24 24" width="28" height="28" fill="currentColor">
        <path d="M12.525.02c1.31-.02 2.61-.01 3.91-.02.08 1.53.63 3.09 1.75 4.17 1.12 1.11 2.7 1.62 4.24 1.79v4.03c-1.44-.05-2.89-.35-4.2-.97-.57-.26-1.1-.59-1.62-.93-.01 2.92.01 5.84-.02 8.75-.08 1.4-.54 2.79-1.35 3.94-1.31 1.92-3.58 3.17-5.91 3.21-1.43.08-2.86-.31-4.08-1.03-2.02-1.19-3.44-3.37-3.65-5.71-.02-.5-.03-1-.01-1.49.18-1.9 1.12-3.72 2.58-4.96 1.66-1.44 3.98-2.13 6.15-1.72.02 1.48-.04 2.96-.04 4.44-.9-.32-1.98-.23-2.81.33-.85.51-1.44 1.43-1.58 2.41-.05.38-.05.77-.01 1.14.12 1.25.96 2.37 2.14 2.78.47.16.97.21 1.46.2.9-.03 1.76-.36 2.44-1.04.72-.73 1.1-1.74 1.11-2.75V0l.04.02z"/>
      </svg>
    </a>

    <!-- REPORT DONATION MODAL -->
    <div v-if="showReportModal" style="position: fixed; top: 0; left: 0; width: 100vw; height: 100vh; background: rgba(0,0,0,0.85); display: flex; align-items: center; justify-content: center; z-index: 9999; padding: 16px;">
      <div style="background: #0d0d18; border: 1px solid #1a1a28; border-radius: 12px; width: 100%; max-width: 420px; padding: 24px; box-shadow: 0 10px 25px rgba(0,0,0,0.5);">
        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px;">
          <h3 style="margin: 0; font-size: 16px; color: #fff; font-weight: 700;">Reportar Donación</h3>
          <button @click="closeReportModal" style="background: none; border: none; color: #606080; cursor: pointer; font-size: 18px; padding: 4px;">&times;</button>
        </div>
        
        <form @submit.prevent="submitDonationReport" style="display: flex; flex-direction: column; gap: 14px;">
          <div style="display: flex; flex-direction: column; gap: 4px;">
            <label style="font-size: 11px; font-weight: 700; color: #606080;">NOMBRE O ALIAS</label>
            <input v-model="reportForm.donorName" type="text" :disabled="reportForm.isAnonymous" placeholder="Ej: Juan Pérez" style="background: #13131f; border: 0.5px solid #2a2a40; border-radius: 6px; padding: 8px 12px; color: #c0c0d8; font-size: 13px;" required />
            <label style="display: inline-flex; align-items: center; gap: 6px; margin-top: 4px; font-size: 12px; color: #9595b0; cursor: pointer;">
              <input type="checkbox" v-model="reportForm.isAnonymous" @change="handleAnonymousChange" />
              Donar de forma anónima
            </label>
          </div>

          <div style="display: flex; flex-direction: column; gap: 4px;">
            <label style="font-size: 11px; font-weight: 700; color: #606080;">MONTO (S/.)</label>
            <input v-model.number="reportForm.amount" type="number" step="0.01" min="0.01" placeholder="Ej: 10.00" style="background: #13131f; border: 0.5px solid #2a2a40; border-radius: 6px; padding: 8px 12px; color: #c0c0d8; font-size: 13px;" required />
          </div>

          <div style="display: flex; flex-direction: column; gap: 4px;">
            <label style="font-size: 11px; font-weight: 700; color: #606080;">MÉTODO DE DONACIÓN</label>
            <select v-model="reportForm.method" style="background: #13131f; border: 0.5px solid #2a2a40; border-radius: 6px; padding: 8px 12px; color: #c0c0d8; font-size: 13px;" required>
              <option value="yape">Yape</option>
              <option value="paypal">PayPal</option>
            </select>
          </div>

          <div style="display: flex; flex-direction: column; gap: 4px;">
            <label style="font-size: 11px; font-weight: 700; color: #606080;">CÓDIGO O NÚMERO DE OPERACIÓN (OPCIONAL)</label>
            <input v-model="reportForm.referenceCode" type="text" placeholder="Ej: 987654 (Ayuda a verificar tu donación)" style="background: #13131f; border: 0.5px solid #2a2a40; border-radius: 6px; padding: 8px 12px; color: #c0c0d8; font-size: 13px;" />
          </div>

          <div style="display: flex; flex-direction: column; gap: 4px;">
            <label style="font-size: 11px; font-weight: 700; color: #606080;">COMPROBANTE O CAPTURA (OPCIONAL)</label>
            <input type="file" @change="handleFileChange" accept="image/*" style="background: #13131f; border: 0.5px solid #2a2a40; border-radius: 6px; padding: 8px 12px; color: #c0c0d8; font-size: 13px;" />
          </div>

          <button type="submit" :disabled="isSubmittingReport" style="background: #00e87a; color: #0d0d18; border: none; border-radius: 6px; padding: 10px; font-size: 13px; font-weight: 700; cursor: pointer; transition: background 0.2s;" onmouseover="this.style.background='#00c969'" onmouseout="this.style.background='#00e87a'">
            {{ isSubmittingReport ? 'Enviando...' : 'Enviar Reporte' }}
          </button>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import VideoPlayer from '../components/VideoPlayer.vue'
import ChatBox from '../components/ChatBox.vue'

const router = useRouter()

const currentUser = ref(null)
const channelInfo = ref(null)
const channelOwner = ref('')
const playbackUrl = ref('')
const playerKey = ref(0)
const viewerCount = ref(0)
const followerCount = ref(0)
const isFollowing = ref(false)
const deferredPrompt = ref(null)
const isStandalone = ref(false)

// ── Cast to TV ────────────────────────────────────────────
const isCasting = ref(false)

const initCastContext = () => {
  if (!window.cast || !window.cast.framework) return
  try {
    const ctx = window.cast.framework.CastContext.getInstance()
    ctx.setOptions({
      receiverApplicationId: window.chrome.cast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
      autoJoinPolicy: window.chrome.cast.AutoJoinPolicy.ORIGIN_SCOPED
    })
    ctx.addEventListener(
      window.cast.framework.CastContextEventType.CAST_STATE_CHANGED,
      (e) => {
        isCasting.value = e.castState === window.cast.framework.CastState.CONNECTED
      }
    )

    // Diagnóstico: observador global del estado del reproductor en el receptor.
    // RemotePlayerController SÍ refleja el estado en todo momento (a diferencia
    // de getMediaSession justo tras loadMedia). Aquí veremos si el receptor
    // queda en BUFFERING, pasa a IDLE con idleReason=ERROR, o llega a PLAYING.
    try {
      const rp = new window.cast.framework.RemotePlayer()
      const rpc = new window.cast.framework.RemotePlayerController(rp)
      const RPET = window.cast.framework.RemotePlayerEventType
      rpc.addEventListener(RPET.PLAYER_STATE_CHANGED, () => {
        console.log('[CAST] playerState:', rp.playerState)
      })
      rpc.addEventListener(RPET.IS_PLAYING_CHANGED, () => {
        console.log('[CAST] isPlaying:', rp.isPlaying, '| duration:', rp.duration, '| currentTime:', rp.currentTime)
      })
      rpc.addEventListener(RPET.MEDIA_INFO_CHANGED, () => {
        console.log('[CAST] mediaInfo:', rp.mediaInfo)
      })
    } catch (e) {
      console.warn('[CAST] no se pudo crear RemotePlayerController:', e)
    }
  } catch (e) {
    console.error('Cast init error:', e)
  }
}

const triggerCastTV = () => {
  if (isCasting.value) {
    const ctx = window.cast?.framework?.CastContext.getInstance()
    if (ctx) ctx.endCurrentSession(true)
    isCasting.value = false
    return
  }

  if (window.cast && window.cast.framework) {
    try {
      const ctx = window.cast.framework.CastContext.getInstance()
      const castState = ctx.getCastState()
      console.log('Cast state before action:', castState)
      
      if (castState === window.cast.framework.CastState.NO_DEVICES_AVAILABLE) {
        console.warn('No Cast devices available on this network')
        alert('No hay dispositivos Cast disponibles en esta red.\nVerifica que estén encendidos y conectados a la misma red WiFi.')
        return
      }
      
      // If already connected, use the existing session
      if (castState === window.cast.framework.CastState.CONNECTED) {
        console.log('Already connected to Cast. Using existing session...')
        const session = typeof ctx.getCurrentSession === 'function'
          ? ctx.getCurrentSession()
          : null
        console.log('Current session from CastContext:', session)
        if (session) {
          loadMediaToSession(session)
          return
        } else {
          console.log('No current session found despite CONNECTED state. Requesting new session...')
        }
      }
      
      // Request a new session
      console.log('Requesting new Cast session...')
      ctx.setOptions({
        receiverApplicationId: window.chrome.cast.media.DEFAULT_MEDIA_RECEIVER_APP_ID,
        autoJoinPolicy: window.chrome.cast.AutoJoinPolicy.ORIGIN_SCOPED
      })
      
      ctx.requestSession().then(
        () => {
          // Pedir la sesión al contexto
          const session = ctx.getCurrentSession()
          console.log('Session obtained:', session)
          if (!session) {
            console.error('Cast session is null')
            alert('No se pudo obtener sesión de Cast.')
            return
          }
          loadMediaToSession(session)
        },
        (err) => {
          console.error('Error requesting cast session:', err?.message)
          alert('Error al conectar con Cast: ' + (err?.message || 'desconocido'))
        }
      )
    } catch (e) {
      console.error('Cast error:', e?.message)
      alert('Error al activar Cast: ' + (e?.message || 'desconocido'))
    }
  } else {
    console.error('Cast API not available')
  }
}

const loadMediaToSession = (session) => {
  if (!session) return
  if (typeof session.loadMedia !== 'function') {
    console.error('Session does not support loadMedia')
    return
  }
  if (!channelInfo.value) return

  // Castear SIEMPRE por el endpoint unificado /api con cast=1, tanto para OBS
  // como para IPTV: sirve un manifiesto plano (secuencia estándar, sin
  // discontinuidades) con el HLS limpio, o el respaldo plano de IPTV. El
  // receptor CAF/Shaka del Chromecast no tolera el manifiesto de secuencias
  // virtuales que sí acepta hls.js en el navegador.
  const castUrl = `${window.location.origin}/api/v1/streams/playback/${channelInfo.value.id}/manifest?cast=1`
  console.log('URL absoluta enviada a la TV:', castUrl)

  const mediaInfo = new window.chrome.cast.media.MediaInfo(castUrl, 'application/x-mpegurl')
  const metadata = new window.chrome.cast.media.GenericMediaMetadata()
  metadata.title = channelInfo.value.name || 'Fuchibol'
  metadata.subtitle = 'En vivo · Fuchibol'
  mediaInfo.metadata = metadata
  mediaInfo.streamType = window.chrome.cast.media.StreamType.LIVE

  // El Default Media Receiver hace "probing" del formato de segmento si no se
  // lo declaramos: reproduce el buffer inicial (~3 segmentos) y luego se queda
  // atascado en BUFFERING al continuar el live. Nuestros segmentos son MPEG-TS
  // (ffmpeg -hls_segment_type mpegts), así que se lo indicamos explícitamente
  // para que continúe recargando el manifiesto y descargando segmentos nuevos.
  if (window.chrome.cast.media.HlsSegmentFormat) {
    mediaInfo.hlsSegmentFormat = window.chrome.cast.media.HlsSegmentFormat.TS
  }
  if (window.chrome.cast.media.HlsVideoSegmentFormat) {
    mediaInfo.hlsVideoSegmentFormat = window.chrome.cast.media.HlsVideoSegmentFormat.MPEG2_TS
  }

  const req = new window.chrome.cast.media.LoadRequest(mediaInfo)
  req.autoplay = true
  session.loadMedia(req).then(
    () => {
      console.log('Media loaded successfully')
      isCasting.value = true

      // Diagnóstico: sondear la sesión de medios y registrar el estado real
      // del receptor (playerState + idleReason) durante ~15s. idleReason=ERROR
      // significa que el receptor rechazó el contenido.
      let tries = 0
      const poll = setInterval(() => {
        tries++
        let media = null
        try { media = session.getMediaSession ? session.getMediaSession() : null } catch (e) { /* noop */ }
        if (media) {
          console.log('[CAST] t=' + tries + ' playerState:', media.playerState,
            '| idleReason:', media.idleReason,
            '| currentTime:', media.getEstimatedTime ? media.getEstimatedTime() : media.currentTime)
        } else {
          console.log('[CAST] t=' + tries + ' sin mediaSession todavía')
        }
        if (tries >= 15) clearInterval(poll)
      }, 1000)
    },
    (err) => console.error('Error loading media:', err)
  )
}

const donationRanking = ref([])
const showReportModal = ref(false)
const isSubmittingReport = ref(false)
const reportForm = ref({
  donorName: '',
  amount: null,
  method: 'yape',
  referenceCode: '',
  isAnonymous: false
})

const handleAnonymousChange = () => {
  if (reportForm.value.isAnonymous) {
    reportForm.value.donorName = 'Anónimo'
  } else {
    reportForm.value.donorName = ''
  }
}

const selectedFile = ref(null)

const handleFileChange = (event) => {
  selectedFile.value = event.target.files[0]
}

const closeReportModal = () => {
  showReportModal.value = false
  selectedFile.value = null
  reportForm.value = {
    donorName: '',
    amount: null,
    method: 'yape',
    referenceCode: '',
    isAnonymous: false
  }
}

const fetchDonationRanking = async () => {
  if (!channelInfo.value) return
  try {
    const res = await fetch(`/api/v1/channels/${channelInfo.value.id}/donations/ranking`)
    if (res.ok) {
      donationRanking.value = await res.json()
    }
  } catch (err) {
    console.error('Error fetching donation ranking:', err)
  }
}

const submitDonationReport = async () => {
  if (isSubmittingReport.value) return
  isSubmittingReport.value = true
  try {
    let receiptUrl = null
    
    // If a receipt screenshot file is selected, upload it first
    if (selectedFile.value) {
      const formData = new FormData()
      formData.append('receipt', selectedFile.value)
      
      const fileRes = await fetch(`/api/v1/channels/${channelInfo.value.id}/donations/receipt`, {
        method: 'POST',
        body: formData
      })
      if (!fileRes.ok) {
        throw new Error('Error al subir comprobante de pago.')
      }
      const fileData = await fileRes.json()
      receiptUrl = fileData.receipt_url
    }

    const res = await fetch(`/api/v1/channels/${channelInfo.value.id}/donations`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        donor_name: reportForm.value.donorName,
        amount: Number(reportForm.value.amount),
        method: reportForm.value.method,
        reference_code: reportForm.value.referenceCode,
        receipt_url: receiptUrl
      })
    })
    if (res.ok) {
      alert('Reporte de donación enviado con éxito. Se mostrará en el ranking una vez sea aprobado por el administrador.')
      closeReportModal()
    } else {
      const data = await res.json()
      alert(data.error || 'Error al enviar reporte de donación.')
    }
  } catch (err) {
    console.error(err)
    alert('Error de conexión al enviar el reporte.')
  } finally {
    isSubmittingReport.value = false
  }
}

const installApp = async () => {
  if (deferredPrompt.value) {
    deferredPrompt.value.prompt()
    const { outcome } = await deferredPrompt.value.userChoice
    if (outcome === 'accepted') {
      deferredPrompt.value = null
    }
  } else {
    alert("Para instalar Fuchibol como App en tu dispositivo:\n\nToca el menú de opciones (los tres puntos o el ícono de Compartir) y selecciona 'Añadir a la pantalla de inicio' o 'Instalar aplicación'.")
  }
}

const shareMatch = async (event) => {
  const text = `Partidos de futbol hoy ⚽\n\n${event.flags || ''} ${event.teams || ''}\n⏰ ${event.time || ''}\n\nMíralo en vivo aquí: ${window.location.href}`
  
  if (navigator.share) {
    try {
      await navigator.share({
        title: 'Fuchibol - En Vivo',
        text: text,
      })
    } catch (err) {
      console.log('Error sharing', err)
    }
  } else {
    navigator.clipboard.writeText(text)
    alert('¡Enlace e información copiados al portapapeles!')
  }
}

const parsedAgendaEvents = computed(() => {
  if (!channelInfo.value || !channelInfo.value.agenda_events) return []
  try {
    const events = JSON.parse(channelInfo.value.agenda_events)
    return Array.isArray(events) ? events.reverse() : []
  } catch (e) {
    return []
  }
})

const handleViewerUpdate = (count) => {
  viewerCount.value = count
}

const handleStreamReload = () => {
  console.log("Señal de stream reload recibida por WS. Recargando reproductor...");
  playerKey.value++
  fetchChannel(true); // Refresca todo forzando el cache busting en la URL
}

const toggleFollow = async () => {
  if (!currentUser.value) {
    router.push('/login')
    return
  }

  const method = isFollowing.value ? 'unfollow' : 'follow'
  try {
    const res = await fetch(`/api/v1/channels/${channelInfo.value.id}/${method}`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${currentUser.value.token}`
      }
    })
    if (res.ok) {
      isFollowing.value = !isFollowing.value
      fetchFollowStatus()
    }
  } catch (err) {
    console.error(err)
  }
}

const fetchFollowStatus = async () => {
  if (!channelInfo.value) return
  try {
    const headers = {}
    if (currentUser.value) {
      headers['Authorization'] = `Bearer ${currentUser.value.token}`
    }
    const res = await fetch(`/api/v1/channels/${channelInfo.value.id}/following`, {
      headers
    })
    if (res.ok) {
      const data = await res.json()
      isFollowing.value = data.is_following
      followerCount.value = data.follower_count
    }
  } catch (err) {
    console.error(err)
  }
}

const logout = () => {
  localStorage.removeItem('fuchibol_user')
  currentUser.value = null
  router.push('/')
}

const fetchChannel = async (forceReload = false) => {
  try {
    const primaryRes = await fetch(`/api/v1/channels/primary`)
    if (!primaryRes.ok) {
      throw new Error('No hay canales configurados')
    }
    const primaryData = await primaryRes.json()
    channelOwner.value = primaryData.username

    const res = await fetch(`/api/v1/channels/by-username/${primaryData.username}`)
    if (!res.ok) {
      throw new Error('Canal no encontrado')
    }
    const data = await res.json()
    
    // Fetch playback info before updating channelInfo to avoid reactivity flapping
    const playRes = await fetch(`/api/v1/streams/playback/${data.id}`)
    if (playRes.ok) {
      const playData = await playRes.json()
      
      // Setup playback URL using webrtc or hls based on stream_type
      let currentUrl = playData.stream_type === 'webrtc' ? playData.webrtc : playData.playback_url;
      if (forceReload && currentUrl && (currentUrl.includes('.m3u8') || currentUrl.includes('/proxy/m3u8'))) {
        const sep = currentUrl.includes('?') ? '&' : '?';
        currentUrl = `${currentUrl}${sep}t=${Date.now()}`;
      }
      
      playbackUrl.value = currentUrl
      data.playback_url = playData.playback_url
      
      data.is_live = playData.is_live
    }

    // Now update channelInfo to trigger a single, accurate reactive update
    channelInfo.value = data
    fetchDonationRanking()

    // Only fetch follow status once or when needed
    if (followerCount.value === 0) {
      fetchFollowStatus()
    }
  } catch (err) {
    console.error(err)
  }
}

let pollInterval = null

onMounted(() => {
  // Verificamos si la app ya está instalada o corriendo en modo standalone
  if (window.matchMedia('(display-mode: standalone)').matches || window.navigator.standalone) {
    isStandalone.value = true
  }

  // Initialize Cast on mount if SDK already loaded
  if (typeof window !== 'undefined') {
    window['__onGCastApiAvailable'] = (isAvailable) => {
      if (isAvailable) initCastContext()
    }
    initCastContext()
  }

  window.addEventListener('beforeinstallprompt', (e) => {
    e.preventDefault()
    deferredPrompt.value = e
  })

  const userObj = localStorage.getItem('fuchibol_user')
  if (userObj) {
    const parsed = JSON.parse(userObj)
    currentUser.value = parsed
    
    // Fetch fresh profile details (including user ID) from backend
    fetch('/api/v1/auth/me', {
      headers: { 'Authorization': `Bearer ${parsed.token}` }
    })
      .then(res => {
        if (res.ok) return res.json()
        throw new Error('Unauthorized')
      })
      .then(userData => {
        currentUser.value = {
          ...parsed,
          id: userData.id,
          username: userData.username,
          role: userData.role
        }
        localStorage.setItem('fuchibol_user', JSON.stringify(currentUser.value))
      })
      .catch(err => {
        console.error('Error fetching user info on mount:', err)
        localStorage.removeItem('fuchibol_user')
        currentUser.value = null
      })
  }
  fetchChannel()
  pollInterval = setInterval(fetchChannel, 10000)
})

const isDonationExpanded = ref(false)

const copyYapeNumber = () => {
  if (channelInfo.value && channelInfo.value.yape_number) {
    navigator.clipboard.writeText(channelInfo.value.yape_number)
    alert('¡Número de Yape copiado al portapapeles!')
  }
}

onBeforeUnmount(() => {
  if (pollInterval) clearInterval(pollInterval)
})
</script>

<style scoped>
/* ── Top Bar ── */
.topbar {
  background: #0d0d18;
  padding: 14px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 0.5px solid #1a1a28;
}
.logo {
  display: flex;
  align-items: center;
  gap: 8px;
}
.logo-dot {
  width: 8px;
  height: 8px;
  background: #00e87a;
  border-radius: 50%;
  animation: pulse-dot 2s infinite;
}
@keyframes pulse-dot {
  0%, 100% { opacity: 1; box-shadow: 0 0 0 0 rgba(0,232,122,0.4); }
  50%       { box-shadow: 0 0 0 6px rgba(0,232,122,0); }
}
.logo-text {
  font-size: 16px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 1px;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.user-badge {
  display: flex;
  align-items: center;
  gap: 6px;
  background: #1a1a2e;
  border: 0.5px solid #2a2a40;
  border-radius: 20px;
  padding: 5px 10px;
}
.user-avatar {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: #00e87a;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 10px;
  font-weight: 700;
  color: #002f18;
}
.user-name {
  font-size: 12px;
  color: #c0c0d8;
}
.exit-btn {
  font-size: 12px;
  color: #606080;
  border: 0.5px solid #2a2a40;
  border-radius: 12px;
  padding: 5px 10px;
  background: none;
  cursor: pointer;
  transition: color 0.2s, border-color 0.2s;
}
.exit-btn:hover { color: #e0e0f0; border-color: #4a4a60; }

/* ── Video Player ── */
.video-wrap {
  position: relative;
  background: #000;
  overflow: hidden;
}
.video-placeholder {
  width: 100%;
  height: 210px;
  background: #080e1a;
  display: flex;
  align-items: center;
  justify-content: center;
}
.video-overlay {
  position: absolute;
  inset: 0;
  background: linear-gradient(to bottom, rgba(0,0,0,0.25) 0%, transparent 40%, rgba(0,0,0,0.65) 100%);
  pointer-events: none;
}
.video-top {
  position: absolute;
  top: 10px;
  left: 12px;
  right: 12px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  z-index: 10;
}
.video-bottom {
  position: absolute;
  bottom: 50px;
  left: 12px;
  z-index: 10;
  pointer-events: none;
}
@media (max-width: 768px) {
  .video-bottom {
    bottom: 60px;
    left: 10px;
  }
}
.live-badge {
  display: flex;
  align-items: center;
  gap: 5px;
  background: #e83d00;
  border-radius: 4px;
  padding: 3px 8px;
}
.live-dot {
  width: 5px;
  height: 5px;
  background: #fff;
  border-radius: 50%;
  animation: pulse-dot 1.4s infinite;
}
.live-text {
  font-size: 11px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 1px;
}
.viewers {
  display: flex;
  align-items: center;
  gap: 5px;
  background: rgba(0,0,0,0.55);
  border-radius: 20px;
  padding: 3px 10px;
}
.viewers-dot {
  width: 6px;
  height: 6px;
  background: #00e87a;
  border-radius: 50%;
}
.viewers-count { font-size: 11px; color: rgba(255,255,255,0.85); font-weight: 500; }

/* ── Match Banner ── */
.match-banner {
  background: #0e1528;
  border: 0.5px solid #1d2540;
  margin: 12px 12px 0;
  border-radius: 10px;
  padding: 10px 14px;
  display: flex;
  align-items: center;
  gap: 10px;
}
.match-flags {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 20px;
}
.match-info { flex: 1; }
.match-title { font-size: 13px; font-weight: 600; color: #e0e0f0; }
.match-time {
  font-size: 13px;
  color: #8c8c9a;
  font-weight: 500;
}

.share-match-btn {
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid rgba(255, 255, 255, 0.1);
  color: #c0c0d8;
  font-size: 10px;
  padding: 4px 8px;
  border-radius: 12px;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 4px;
  transition: all 0.2s ease;
  text-transform: uppercase;
  font-weight: 600;
  letter-spacing: 0.5px;
}

.share-match-btn:hover {
  background: rgba(0, 232, 122, 0.15);
  border-color: rgba(0, 232, 122, 0.4);
  color: #00e87a;
}

.match-pill {
  background: rgba(0,232,122,0.1);
  border: 0.5px solid rgba(0,232,122,0.3);
  border-radius: 20px;
  padding: 3px 10px;
  font-size: 10px;
  color: #00e87a;
  font-weight: 700;
  letter-spacing: 0.5px;
  white-space: nowrap;
}

/* ── Channel Row ── */
.channel-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  background: #0d0d18;
  margin: 12px 12px 0;
  border-radius: 10px;
  border: 0.5px solid #1a1a28;
}
.channel-icon {
  width: 46px;
  height: 46px;
  background: #001a0d;
  border: 0.5px solid rgba(0,232,122,0.2);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: #00e87a;
  letter-spacing: 0.5px;
  flex-shrink: 0;
  object-fit: cover;
}
.channel-info { flex: 1; min-width: 0; }
.channel-name { font-size: 13px; font-weight: 600; color: #e0e0f0; }
.channel-meta {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 3px;
  flex-wrap: wrap;
}
.channel-handle   { font-size: 11px; color: #505070; }
.channel-followers{ font-size: 11px; color: #505070; }
.channel-live-tag {
  background: #e83d00;
  border-radius: 3px;
  padding: 1px 6px;
  font-size: 9px;
  font-weight: 700;
  color: #fff;
  letter-spacing: 0.5px;
}
.follow-btn {
  background: transparent;
  border: 1.5px solid #00e87a;
  border-radius: 20px;
  padding: 6px 14px;
  font-size: 12px;
  font-weight: 600;
  color: #00e87a;
  cursor: pointer;
  transition: background 0.2s, color 0.2s;
  white-space: nowrap;
}
.follow-btn:hover { background: rgba(0,232,122,0.1); }

/* ── Section Label ── */
.section-label {
  font-size: 10px;
  font-weight: 700;
  color: #404060;
  letter-spacing: 1.5px;
  padding: 14px 16px 4px;
}
.streamer-bio {
  font-size: 13px;
  color: #8080a0;
  padding: 4px 16px 14px;
  line-height: 1.6;
}

/* Floating Buttons */
.wsp-float-btn,
.tiktok-float-btn {
  position: fixed;
  right: 20px;
  width: 50px;
  height: 50px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  box-shadow: 0 4px 10px rgba(0, 0, 0, 0.3);
  transition: transform 0.2s;
  z-index: 1000;
}

.wsp-float-btn {
  bottom: 20px;
  background-color: #25D366;
}

.tiktok-float-btn {
  bottom: 80px;
  background-color: #000000;
  border: 1px solid rgba(255,255,255,0.2);
}

.wsp-float-btn:hover,
.tiktok-float-btn:hover {
  transform: scale(1.1);
}

.hide-desktop {
  display: block;
}
.hide-mobile {
  display: none;
}

@media (min-width: 1024px) {
  .hide-desktop {
    display: none !important;
  }
  .hide-mobile {
    display: block !important;
  }
  .home-grid {
    display: grid;
    grid-template-columns: 7fr 3fr;
    gap: 0;
    align-items: stretch;
  }
  .home-main {
    border-right: 0.5px solid #1a1a28;
    display: flex;
    flex-direction: column;
  }
  .home-sidebar {
    display: flex;
    flex-direction: column;
    padding-top: 16px;
  }
  .video-wrap {
    height: auto;
    aspect-ratio: 16/9;
  }
}

@media (max-width: 480px) {
  .topbar { padding: 10px 8px; }
  .logo-text { font-size: 14px; }
  .user-badge { padding: 4px 6px; }
  .user-name { display: none; } /* Hide username on very small screens to save space */
  .follow-btn, .exit-btn { font-size: 10px; padding: 4px 8px; }
  .channel-row { flex-wrap: wrap; justify-content: center; text-align: center; }
  .channel-meta { justify-content: center; }
  .follow-btn { width: 100%; margin-top: 10px; }
}

.cast-tv-btn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  background: rgba(0, 0, 0, 0.5);
  color: #c0c0d8;
  border: 0.5px solid #2a2a40;
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  backdrop-filter: blur(4px);
  transition: all 0.2s;
}

.cast-tv-btn:hover {
  background: rgba(0, 232, 122, 0.1);
  border-color: rgba(0, 232, 122, 0.35);
  color: #00e87a;
}

.cast-tv-btn.active {
  background: rgba(0, 232, 122, 0.15);
  border-color: #00e87a;
  color: #00e87a;
}
</style>